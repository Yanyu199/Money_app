class FundModel {
  // --- 基础信息 ---
  final String fundCode;
  final String name;
  final String jzrq; // 净值日期 (新增)
  final String dwjz; // 单位净值 (新增)
  final String gsz;  // 估算值 (实时净值)
  final String gszzl; // 估算增长率 (+1.25%)
  final String gzTime; // 更新时间
  final String premiumRate; // 🔥 新增：溢价率 (String类型，例如 "+1.2%")

  // --- 持仓数值 (上一轮修复崩溃必须的字段) ---
  final double shares;      // 持仓份额
  final double costPrice;   // 成本价
  final double totalValue;  // 市值
  final double totalReturn; // 持有收益
  final double dayReturn;   // 当日收益

  FundModel({
    required this.fundCode,
    required this.name,
    this.jzrq = "",
    this.dwjz = "",
    required this.gsz,
    required this.gszzl,
    required this.gzTime,
    this.premiumRate = "", // 默认为空字符串
    
    // 数值字段初始化
    required this.shares,
    required this.costPrice,
    required this.totalValue,
    required this.totalReturn,
    required this.dayReturn,
  });

  // 工厂方法：从 JSON 创建对象
  factory FundModel.fromJson(Map<String, dynamic> json) {
    // 辅助函数：安全地将 JSON 中的数字转为 double
    double parseDouble(dynamic value) {
      if (value == null) return 0.0;
      if (value is int) return value.toDouble();
      if (value is double) return value;
      if (value is String) return double.tryParse(value) ?? 0.0;
      return 0.0;
    }

    return FundModel(
      fundCode: json['fund_code'] ?? json['fundcode'] ?? '', // 兼容不同写法
      name: json['fund_name'] ?? json['name'] ?? '未知基金',
      
      // 新增字段解析
      jzrq: json['jzrq'] ?? "",
      dwjz: json['dwjz'] ?? "",
      premiumRate: json['premium_rate'] ?? "", 

      // 原有字段
      gsz: json['last_price'] ?? json['gsz'] ?? '0.00',
      gszzl: json['change'] ?? json['gszzl'] ?? '0.00',
      gzTime: json['gztime'] ?? '--:--',
      
      // 🔥 数值字段 (防止 null 崩溃)
      shares: parseDouble(json['shares']),
      costPrice: parseDouble(json['cost_price']),
      totalValue: parseDouble(json['total_value']),
      totalReturn: parseDouble(json['total_return']),
      dayReturn: parseDouble(json['day_return']),
    );
  }
}