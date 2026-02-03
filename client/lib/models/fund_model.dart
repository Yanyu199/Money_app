class FundModel {
  final String fundCode;
  final String name;
  final String jzrq; // 净值日期
  final String dwjz; // 单位净值
  final String gsz;  // 估算值
  final String gszzl; // 估算涨跌幅
  final String gzTime; // 估值时间
  final String premiumRate; // 🔥 新增：溢价率

  FundModel({
    required this.fundCode,
    required this.name,
    required this.jzrq,
    required this.dwjz,
    required this.gsz,
    required this.gszzl,
    required this.gzTime,
    this.premiumRate = "",
  });

  factory FundModel.fromJson(Map<String, dynamic> json) {
    return FundModel(
      fundCode: json['fundcode'] ?? "",
      name: json['name'] ?? "",
      jzrq: json['jzrq'] ?? "",
      dwjz: json['dwjz'] ?? "",
      gsz: json['gsz'] ?? "",
      gszzl: json['gszzl'] ?? "",
      gzTime: json['gztime'] ?? "",
      premiumRate: json['premium_rate'] ?? "", // 解析新字段
    );
  }
}