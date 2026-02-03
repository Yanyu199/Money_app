import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

class HttpService {
  static String baseUrl = "https://replace-me.cpolar.top";
  static String? _token; 

  static Future<void> initBaseUrl() async {
    final prefs = await SharedPreferences.getInstance();
    final savedUrl = prefs.getString('api_base_url');
    if (savedUrl != null && savedUrl.isNotEmpty) baseUrl = savedUrl;
  }

  static Future<void> setBaseUrl(String newUrl) async {
    if (newUrl.endsWith("/")) newUrl = newUrl.substring(0, newUrl.length - 1);
    if (!newUrl.startsWith("http")) newUrl = "https://$newUrl";
    baseUrl = newUrl;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('api_base_url', newUrl);
  }

  // ... 登录注册逻辑保持不变 ...
  static Future<bool> login(String username, String password) async {
      try {
        final response = await http.post(
          Uri.parse('$baseUrl/login'),
          body: jsonEncode({'username': username, 'password': password}),
        );
        if (response.statusCode == 200) {
          final data = jsonDecode(response.body);
          _token = data['token'];
          final prefs = await SharedPreferences.getInstance();
          await prefs.setString('token', _token!);
          return true;
        }
      } catch (e) {}
      return false;
  }
  
  static Future<bool> register(String username, String password) async {
     try {
       final response = await http.post(Uri.parse('$baseUrl/register'), body: jsonEncode({'username': username, 'password': password}));
       return response.statusCode == 200;
     } catch (e) { return false; }
  }

  static Future<bool> tryAutoLogin() async {
    final prefs = await SharedPreferences.getInstance();
    if (prefs.containsKey('token')) { _token = prefs.getString('token'); return true; }
    return false;
  }
  static Future<void> logout() async { _token = null; final prefs = await SharedPreferences.getInstance(); await prefs.remove('token'); }

  static Future<Map<String, dynamic>?> getMyData() async {
    if (_token == null) return null;
    try {
      final response = await http.get(Uri.parse('$baseUrl/my_data'), headers: {'Authorization': _token!});
      if (response.statusCode == 200) return jsonDecode(utf8.decode(response.bodyBytes));
    } catch (e) {}
    return null;
  }

  static Future<List<dynamic>> refreshMarketData() async {
    if (_token == null) return [];
    try {
      final response = await http.get(Uri.parse('$baseUrl/refresh_market'), headers: {'Authorization': _token!});
      if (response.statusCode == 200) {
        final json = jsonDecode(utf8.decode(response.bodyBytes));
        return json['data'] as List<dynamic>;
      }
    } catch (e) {}
    return [];
  }

  static Future<bool> addFundDB(String code, String type, double amount) async {
    if (_token == null) return false;
    try {
      final response = await http.post(
        Uri.parse('$baseUrl/add'),
        headers: {'Authorization': _token!},
        body: jsonEncode({'code': code, 'type': type, 'amount': amount}),
      );
      return response.statusCode == 200;
    } catch (e) { return false; }
  }

  static Future<bool> deleteFund(String code, String type) async {
    if (_token == null) return false;
    try {
      final response = await http.post(
        Uri.parse('$baseUrl/delete'),
        headers: {'Authorization': _token!},
        body: jsonEncode({'code': code, 'type': type}),
      );
      return response.statusCode == 200;
    } catch (e) { return false; }
  }

  static Future<Map<String, dynamic>?> getDetail(String code) async {
    try {
      final response = await http.get(Uri.parse('$baseUrl/detail?code=$code'));
      if (response.statusCode == 200) return jsonDecode(utf8.decode(response.bodyBytes));
    } catch (e) {}
    return null;
  }

  static Future<List<dynamic>> searchFund(String keyword) async {
    if (_token == null) return [];
    try {
      final response = await http.get(Uri.parse('$baseUrl/search?key=$keyword'), headers: {'Authorization': _token!});
      if (response.statusCode == 200) {
        final json = jsonDecode(utf8.decode(response.bodyBytes));
        return json['data'] as List<dynamic>;
      }
    } catch (e) {}
    return [];
  }

  // 🔥 新增：调用结算接口
  static Future<String> settleHoldings() async {
    if (_token == null) return "未登录";
    try {
      final response = await http.post(
        Uri.parse('$baseUrl/settle'),
        headers: {'Authorization': _token!},
      );
      if (response.statusCode == 200) {
        final data = jsonDecode(utf8.decode(response.bodyBytes));
        return data['message']; 
      }
    } catch (e) {
      return "网络错误: $e";
    }
    return "结算失败";
  }
}