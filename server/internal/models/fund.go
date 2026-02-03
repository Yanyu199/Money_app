package models

// FundInfo 基础行情信息
type FundInfo struct {
	FundCode    string `json:"fundcode"`
	Name        string `json:"name"`
	JZ          string `json:"jz"`
	GSZ         string `json:"gsz"`
	GSZZL       string `json:"gszzl"`
	GZTime      string `json:"gztime"`
	PremiumRate string `json:"premium_rate"`
}

// 🔥 新增：股票详细信息
type StockInfo struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Price  string `json:"price"`  // 实时股价
	Change string `json:"change"` // 实时涨跌幅
}

// FundDetail 基金详情
type FundDetail struct {
	FundCode     string      `json:"fundcode"`
	Stocks       []string    `json:"stocks"`        // 保留旧字段兼容
	StockDetails []StockInfo `json:"stock_details"` // 🔥 新增：重仓股详情列表
	Sectors      []string    `json:"sectors"`
}

// 搜索结果模型
type FundSearchResult struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}
