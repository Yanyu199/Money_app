import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/fund_model.dart';

class SocketService {
  late WebSocketChannel _channel;
  
  // 用于通知 UI 更新的回调函数
  final Function(FundModel) onDataReceived;

  SocketService({required this.onDataReceived});

  void connect() {
    // ⚠️ 注意：Android 模拟器访问本机 localhost 需要用 10.0.2.2
    // 如果是真机，请填写电脑的局域网 IP (例如 ws://192.168.1.5:8080/ws)
    final wsUrl = Uri.parse('ws://10.0.2.2:8080/ws'); 
    
    print("正在连接 WebSocket: $wsUrl");
    _channel = WebSocketChannel.connect(wsUrl);

    _channel.stream.listen(
      (message) {
        print("收到数据: $message");
        try {
          final Map<String, dynamic> jsonMap = jsonDecode(message);
          // 如果是 Ping 消息则忽略，如果是数据则解析
          if (jsonMap.containsKey('fundcode')) {
            final fund = FundModel.fromJson(jsonMap);
            onDataReceived(fund);
          }
        } catch (e) {
          print("解析错误: $e"); // 忽略非 JSON 消息 (如 ping)
        }
      },
      onError: (error) => print("WS 错误: $error"),
      onDone: () => print("WS 连接断开"),
    );
  }

  void dispose() {
    _channel.sink.close();
  }
}