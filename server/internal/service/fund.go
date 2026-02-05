package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"fund-tracker-server/internal/models"
	"fund-tracker-server/pkg/utils"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// FetchFundData 智能混合模式
func FetchFundData(code string) (*models.FundInfo, error) {
	var finalFund *models.FundInfo

	if isExchangeTraded(code) {
		marketFund, err := fetchMarketData(code)
		if err == nil && marketFund != nil {
			finalFund = marketFund
			finalFund.GZTime += " (实时)"

			gzFund, _ := fetchEstimateData(code)
			if gzFund == nil {
				gzFund, _ = fetchFinalData(code)
			}

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

	if finalFund == nil {
		gzFund, err := fetchEstimateData(code)
		if err == nil && gzFund != nil {
			finalFund = gzFund
		}
	}

	f10Fund, err := fetchFinalData(code)

	if finalFund == nil {
		if f10Fund != nil {
			f10Fund.GZTime += " (确)"
			return f10Fund, nil
		}
		return nil, fmt.Errorf("无数据: %v", err)
	}

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

	if isQDII(finalFund.Name) {
		status := getUSMarketStatus()
		if status != "" {
			finalFund.GZTime += " " + status
		}
	}

	return finalFund, nil
}

// SearchFund 模糊搜索
func SearchFund(keyword string) ([]models.FundSearchResult, error) {
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

// FetchFundDetail 基金详情
func FetchFundDetail(code string) (*models.FundDetail, error) {
	urlBase := fmt.Sprintf("https://fundmobapi.eastmoney.com/FundMNewApi/FundMNBasicInformation?FCODE=%s&deviceid=123&plat=Iphone&product=EFund&version=6.0.0", code)
	body, err := httpGet(urlBase)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	var stocks []string
	var baseStockList []models.StockInfo
	var secids []string

	if result != nil && result["Datas"] != nil {
		datas, ok := result["Datas"].(map[string]interface{})
		if ok {
			if stockList, ok := datas["InverstPositionList"].([]interface{}); ok {
				for _, s := range stockList {
					if item, ok := s.(map[string]interface{}); ok {
						name := item["GPNM"].(string)
						code := item["GPDM"].(string)
						stocks = append(stocks, name)
						baseStockList = append(baseStockList, models.StockInfo{
							Name: name, Code: code, Price: "--", Change: "--",
						})
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

	if len(secids) > 0 {
		api := fmt.Sprintf("http://push2.eastmoney.com/api/qt/ulist.np/get?secids=%s&fields=f12,f14,f2,f3", strings.Join(secids, ","))
		body, err := httpGet(api)
		if err == nil {
			var stockRes struct {
				Data struct {
					Diff []struct {
						F12 string  `json:"f12"`
						F2  float64 `json:"f2"`
						F3  float64 `json:"f3"`
					} `json:"diff"`
				} `json:"data"`
			}
			json.Unmarshal(body, &stockRes)
			realTimeMap := make(map[string]struct {
				Price  string
				Change string
			})
			for _, item := range stockRes.Data.Diff {
				priceStr := fmt.Sprintf("%.2f", item.F2)
				changeStr := fmt.Sprintf("%+.2f%%", item.F3)
				if item.F2 == 0 {
					priceStr = "--"
					changeStr = "--"
				}
				realTimeMap[item.F12] = struct {
					Price  string
					Change string
				}{priceStr, changeStr}
			}
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
		StockDetails: baseStockList,
		Sectors:      sectors,
	}, nil
}

// ---------------- 内部 Helper 函数 ----------------

func isExchangeTraded(code string) bool {
	return strings.HasPrefix(code, "15") ||
		strings.HasPrefix(code, "16") ||
		strings.HasPrefix(code, "51") ||
		strings.HasPrefix(code, "56") ||
		strings.HasPrefix(code, "58")
}

func isQDII(name string) bool {
	keywords := []string{"标普", "纳斯达克", "美国", "海外", "QDII", "全球", "恒生", "港股"}
	for _, kw := range keywords {
		if strings.Contains(name, kw) {
			return true
		}
	}
	return false
}

func getUSMarketStatus() string {
	now := time.Now()
	hour := now.Hour()
	minute := now.Minute()
	isTrading := (hour == 21 && minute >= 30) || (hour > 21) || (hour < 4)
	if isTrading {
		if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
			return "[美股休市]"
		}
		return "[美股交易中]"
	}
	return ""
}

func fetchMarketData(code string) (*models.FundInfo, error) {
	market := "0"
	if strings.HasPrefix(code, "5") || strings.HasPrefix(code, "6") {
		market = "1"
	}
	url := fmt.Sprintf("http://push2.eastmoney.com/api/qt/stock/get?secid=%s.%s&fields=f43,f57,f58,f169,f170,f46,f60", market, code)
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	var result struct {
		Data *struct {
			F43  float64 `json:"f43"`
			F60  float64 `json:"f60"`
			F170 float64 `json:"f170"`
			F58  string  `json:"f58"`
		} `json:"data"`
	}
	json.Unmarshal(body, &result)
	if result.Data == nil {
		return nil, fmt.Errorf("no market data")
	}
	price := result.Data.F43
	if price <= 0.0001 {
		price = result.Data.F60
	}
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

func fetchEstimateData(code string) (*models.FundInfo, error) {
	url := fmt.Sprintf("http://fundgz.1234567.com.cn/js/%s.js?rt=%d", code, time.Now().Unix())
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	jsonString := utils.ParseJSONP(string(body))
	if jsonString == "" {
		return nil, fmt.Errorf("empty jsonp")
	}
	var fund models.FundInfo
	json.Unmarshal([]byte(jsonString), &fund)
	return &fund, nil
}

func fetchFinalData(code string) (*models.FundInfo, error) {
	url := fmt.Sprintf("http://fund.eastmoney.com/f10/F10DataApi.aspx?type=lsjz&code=%s&page=1&per=1", code)
	body, err := httpGet(url)
	if err != nil {
		return nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	firstRow := doc.Find("tbody tr").First()
	if firstRow.Length() == 0 {
		return nil, fmt.Errorf("no data")
	}
	tds := firstRow.Find("td")
	if tds.Length() < 4 {
		return nil, fmt.Errorf("table error")
	}
	return &models.FundInfo{
		GZTime: tds.Eq(0).Text(),
		GSZ:    tds.Eq(1).Text(),
		GSZZL:  strings.ReplaceAll(tds.Eq(3).Text(), "%", ""),
	}, nil
}

// 通用 HTTP GET 请求 (🔥 优化：5秒超时)
func httpGet(url string) ([]byte, error) {
	// 🔥 Timeout 设置为 5 秒
	client := http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "http://fund.eastmoney.com/")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
