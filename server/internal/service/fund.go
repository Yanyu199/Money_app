package service

import (
	"encoding/json"
	"fmt"
	"fund-tracker-server/internal/models"
	"fund-tracker-server/pkg/utils"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FetchFundData 智能混合模式：场内基金(LOF/ETF)优先看实时行情，场外基金看估值，并支持计算溢价率
func FetchFundData(code string) (*models.FundInfo, error) {
	var finalFund *models.FundInfo

	// 1. 尝试获取场内实时行情 (LOF/ETF)
	if isExchangeTraded(code) {
		marketFund, err := fetchMarketData(code)
		if err == nil && marketFund != nil {
			finalFund = marketFund
			finalFund.GZTime += " (实时)"

			// 🔥 计算溢价率：(市价 - 估值) / 估值
			// 需要再抓一次估值数据来对比
			gzFund, _ := fetchEstimateData(code)
			if gzFund != nil && gzFund.GSZ != "" {
				price, _ := strconv.ParseFloat(finalFund.GSZ, 64)
				nav, _ := strconv.ParseFloat(gzFund.GSZ, 64)
				if nav > 0 {
					rate := (price - nav) / nav * 100
					finalFund.PremiumRate = fmt.Sprintf("%+.2f%%", rate)
				}
			}
		}
	}

	// 2. 如果不是场内，或者场内没取到，取普通估值
	if finalFund == nil {
		gzFund, err := fetchEstimateData(code)
		if err == nil && gzFund != nil {
			finalFund = gzFund
		}
	}

	// 3. 兜底逻辑：获取 F10 净值 (官方确权数据)
	// 比如晚上或者周末，估值接口可能停了，用这个作为参考
	f10Fund, _ := fetchFinalData(code) // 这里忽略 err，因为只是兜底

	if finalFund == nil {
		if f10Fund != nil {
			f10Fund.GZTime += " (确)"
			return f10Fund, nil
		}
		return nil, fmt.Errorf("无数据")
	}

	// 4. 时间比对优化 (如果 F10 更确切，覆盖估值)
	if f10Fund != nil {
		gzDateStr := strings.Split(finalFund.GZTime, " ")[0]
		if f10Fund.GZTime >= gzDateStr && !strings.Contains(finalFund.GZTime, "实时") {
			finalFund.GSZ = f10Fund.GSZ
			finalFund.GSZZL = f10Fund.GSZZL
			finalFund.GZTime = f10Fund.GZTime + " (确)"
		} else if !strings.Contains(finalFund.GZTime, "实时") {
			finalFund.GZTime += " (估)"
		}
	} else if !strings.Contains(finalFund.GZTime, "实时") {
		finalFund.GZTime += " (估)"
	}

	return finalFund, nil
}

// SearchFund 模糊搜索基金
func SearchFund(keyword string) ([]models.FundSearchResult, error) {
	// 东方财富搜索接口
	api := fmt.Sprintf("http://fundsuggest.eastmoney.com/FundSearch/api/FundSearchAPI.ashx?m=1&key=%s", url.QueryEscape(keyword))
	body, err := httpGet(api)
	if err != nil {
		return nil, err
	}

	var result struct {
		Datas []struct {
			CODE         string `json:"CODE"`
			NAME         string `json:"NAME"`
			CATEGORYDESC string `json:"CATEGORYDESC"`
		} `json:"Datas"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	var list []models.FundSearchResult
	for _, item := range result.Datas {
		list = append(list, models.FundSearchResult{
			Code: item.CODE,
			Name: item.NAME,
			Type: item.CATEGORYDESC,
		})
	}
	return list, nil
}

// FetchFundDetail 获取基金详情（双重保障：先获取静态名单，再尝试填充实时行情）
func FetchFundDetail(code string) (*models.FundDetail, error) {
	// 1. 获取基金基础信息 (这是数据的“骨架”，必须有)
	urlBase := fmt.Sprintf("https://fundmobapi.eastmoney.com/FundMNewApi/FundMNBasicInformation?FCODE=%s&deviceid=123&plat=Iphone&product=EFund&version=6.0.0", code)
	body, err := httpGet(urlBase)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	var stocks []string                  // 仅用于兼容旧版前端
	var baseStockList []models.StockInfo // 存基础的 {代码, 名称}
	var secids []string                  // 用于请求实时的 ID 列表

	if result != nil && result["Datas"] != nil {
		datas, ok := result["Datas"].(map[string]interface{})
		if ok {
			if stockList, ok := datas["InverstPositionList"].([]interface{}); ok {
				for _, s := range stockList {
					if item, ok := s.(map[string]interface{}); ok {
						name := item["GPNM"].(string) // 股票名称
						code := item["GPDM"].(string) // 股票代码

						stocks = append(stocks, name)

						// 先把“骨架”存起来，默认价格是 "--"
						baseStockList = append(baseStockList, models.StockInfo{
							Name:   name,
							Code:   code,
							Price:  "--",
							Change: "--",
						})

						// 判断市场 ID (用于下一步查实时)
						// 1: 上海 (6开头)
						// 0: 深圳 (0/3开头), 北交所 (4/8开头)
						market := "0"
						if strings.HasPrefix(code, "6") {
							market = "1"
						}
						secids = append(secids, fmt.Sprintf("%s.%s", market, code))
					}
				}
			}
		}
	}

	// 2. 尝试获取实时行情 (这是数据的“血肉”，可能获取失败)
	if len(secids) > 0 {
		api := fmt.Sprintf("http://push2.eastmoney.com/api/qt/ulist.np/get?secids=%s&fields=f12,f14,f2,f3", strings.Join(secids, ","))
		body, err := httpGet(api)
		if err == nil {
			var stockRes struct {
				Data struct {
					Diff []struct {
						F12 string  `json:"f12"` // 代码
						F2  float64 `json:"f2"`  // 最新价
						F3  float64 `json:"f3"`  // 涨跌幅
					} `json:"diff"`
				} `json:"data"`
			}
			json.Unmarshal(body, &stockRes)

			// 将实时数据转为 Map，方便查找
			realTimeMap := make(map[string]struct {
				Price  string
				Change string
			})
			for _, item := range stockRes.Data.Diff {
				// 格式化数据
				priceStr := fmt.Sprintf("%.2f", item.F2)
				changeStr := fmt.Sprintf("%+.2f%%", item.F3)
				// 如果价格是 0 (休市或停牌)，显示 "--"
				if item.F2 == 0 {
					priceStr = "--"
					changeStr = "--"
				}

				realTimeMap[item.F12] = struct {
					Price  string
					Change string
				}{priceStr, changeStr}
			}

			// 3. 将实时数据填入骨架
			for i := range baseStockList {
				if val, ok := realTimeMap[baseStockList[i].Code]; ok {
					baseStockList[i].Price = val.Price
					baseStockList[i].Change = val.Change
				}
			}
		}
	}

	sectors := []string{"关联持仓行业"}
	if len(baseStockList) == 0 {
		sectors = []string{"暂无持仓数据"}
	}

	return &models.FundDetail{
		FundCode:     code,
		Stocks:       stocks,
		StockDetails: baseStockList, // 无论是否有实时数据，这里都有值
		Sectors:      sectors,
	}, nil
}

// ---------------- 内部函数 ----------------

// 判断是否为场内基金 (LOF/ETF)
func isExchangeTraded(code string) bool {
	return strings.HasPrefix(code, "15") || // 深圳 ETF/LOF
		strings.HasPrefix(code, "16") || // 深圳 LOF (如 161226, 162411)
		strings.HasPrefix(code, "51") || // 上海 ETF
		strings.HasPrefix(code, "56") || // 上海 ETF
		strings.HasPrefix(code, "58") // 上海 ETF
}

// 获取场内实时行情
func fetchMarketData(code string) (*models.FundInfo, error) {
	// 0: 深圳 (15xxx, 16xxx)
	// 1: 上海 (51xxx, 56xxx, 58xxx, 6xxxxx)
	market := "0"
	if strings.HasPrefix(code, "5") || strings.HasPrefix(code, "6") {
		market = "1"
	}

	// 请求东方财富股票行情接口
	// f43: 最新价, f60: 昨收, f170: 涨跌幅%
	url := fmt.Sprintf("http://push2.eastmoney.com/api/qt/stock/get?secid=%s.%s&fields=f43,f57,f58,f169,f170,f46,f60", market, code)

	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data *struct {
			F43  float64 `json:"f43"`  // 最新价
			F60  float64 `json:"f60"`  // 昨收 (兜底用)
			F170 float64 `json:"f170"` // 涨跌幅%
			F58  string  `json:"f58"`  // 名称
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Data == nil {
		return nil, fmt.Errorf("no market data")
	}

	price := result.Data.F43
	// 如果最新价是 0 (可能是停牌、未开盘)，强制使用昨收价 f60
	if price <= 0.0001 {
		price = result.Data.F60
	}
	// 如果还是 0，说明真的没数据
	if price <= 0.0001 {
		return nil, fmt.Errorf("price is zero")
	}

	return &models.FundInfo{
		FundCode: code,
		Name:     result.Data.F58,
		GSZ:      fmt.Sprintf("%.3f", price),
		GSZZL:    fmt.Sprintf("%.2f", result.Data.F170),
		GZTime:   time.Now().Format("2006-01-02 15:04"),
	}, nil
}

// 获取场外基金估值 (js 接口)
func fetchEstimateData(code string) (*models.FundInfo, error) {
	url := fmt.Sprintf("http://fundgz.1234567.com.cn/js/%s.js?rt=%d", code, time.Now().Unix())
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	jsonString := utils.ParseJSONP(string(body))
	if jsonString == "" {
		return nil, fmt.Errorf("empty")
	}
	var fund models.FundInfo
	json.Unmarshal([]byte(jsonString), &fund)
	return &fund, nil
}

// 获取场外基金最终净值 (HTML 解析)
func fetchFinalData(code string) (*models.FundInfo, error) {
	url := fmt.Sprintf("http://fund.eastmoney.com/f10/F10DataApi.aspx?type=lsjz&code=%s&page=1&per=1", code)
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`<tbody>\s*<tr>\s*<td>(.*?)</td>\s*<td.*?>(.*?)</td>\s*<td.*?>(.*?)</td>\s*<td.*?>(.*?)%?</td>`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 5 {
		return nil, fmt.Errorf("fail")
	}
	return &models.FundInfo{GZTime: matches[1], GSZ: matches[2], GSZZL: matches[4]}, nil
}

// 通用 HTTP GET 请求
func httpGet(url string) ([]byte, error) {
	client := http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	// 伪装浏览器 Header，防止反爬
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
