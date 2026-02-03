import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/fund_model.dart';

class SocketService {
  late WebSocketChannel _channel;
  final Function(FundModel) onDataReceived;

  SocketService({required this.onDataReceived});

  void connect() {
    // 连接本机服务
    final wsUrl = Uri.parse('wss://40ce835e.r34.cpolar.top/ws'); 
    
    print("正在连接: $wsUrl");
    _channel = WebSocketChannel.connect(wsUrl);

    _channel.stream.listen(
      (message) {
        try {
          // 忽略 ping 消息
          if (message == "ping") return;

          final Map<String, dynamic> jsonMap = jsonDecode(message);
          if (jsonMap.containsKey('fundcode')) {
            onDataReceived(FundModel.fromJson(jsonMap));
          }
        } catch (e) {
          print("数据解析忽略: $message");
        }
      },
      onError: (error) => print("WS 错误: $error"),
    );
  }

  void dispose() {
    _channel.sink.close();
  }
}