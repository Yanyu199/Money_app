class FundModel {
  final String fundCode;
  final String name;
  final String gsz; // 估算值 (实时净值)
  final String gszzl; // 估算增长率 (+1.25%)
  final String gzTime; // 更新时间

  FundModel({
    required this.fundCode,
    required this.name,
    required this.gsz,
    required this.gszzl,
    required this.gzTime,
  });

  // 工厂方法：从 JSON 创建对象
  factory FundModel.fromJson(Map<String, dynamic> json) {
    return FundModel(
      fundCode: json['fundcode'] ?? '',
      name: json['name'] ?? '未知基金',
      gsz: json['gsz'] ?? '0.00',
      gszzl: json['gszzl'] ?? '0.00',
      gzTime: json['gztime'] ?? '--:--',
    );
  }
}