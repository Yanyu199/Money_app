import 'package:flutter/material.dart';
import '../api/socket_service.dart';
import '../models/fund_model.dart';
import 'dart:async';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  FundModel? _currentFund;
  late SocketService _socketService;
  String _status = "等待连接...";

  @override
  void initState() {
    super.initState();
    // 初始化 WebSocket
    _socketService = SocketService(onDataReceived: (fund) {
      if (mounted) {
        setState(() {
          _currentFund = fund;
          _status = "实时监控中";
        });
      }
    });
    
    // 延时 1 秒启动连接，确保页面加载完成
    Timer(const Duration(seconds: 1), () {
      _socketService.connect();
    });
  }

  @override
  void dispose() {
    _socketService.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.grey[100],
      appBar: AppBar(
        title: const Text("FundTracker 实时监控"),
        backgroundColor: Colors.blueAccent,
        foregroundColor: Colors.white,
      ),
      body: Center(
        child: _currentFund == null
            ? Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const CircularProgressIndicator(),
                  const SizedBox(height: 20),
                  Text(_status, style: const TextStyle(color: Colors.grey)),
                ],
              )
            : _buildFundCard(),
      ),
    );
  }

  Widget _buildFundCard() {
    final fund = _currentFund!;
    // 判断涨跌颜色：涨(>0)为红，跌(<0)为绿 (符合A股习惯)
    final isUp = double.tryParse(fund.gszzl)?.isNegative == false;
    final color = isUp ? Colors.red : Colors.green;

    return Container(
      margin: const EdgeInsets.all(20),
      padding: const EdgeInsets.all(25),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(20),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.1),
            blurRadius: 20,
            offset: const Offset(0, 10),
          )
        ],
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            fund.name,
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 10),
          Text(
            fund.fundCode,
            style: TextStyle(color: Colors.grey[600], fontSize: 14),
          ),
          const Divider(height: 40),
          const Text("实时估值", style: TextStyle(color: Colors.grey)),
          const SizedBox(height: 5),
          Text(
            fund.gsz,
            style: TextStyle(
              fontSize: 48,
              fontWeight: FontWeight.bold,
              color: color,
            ),
          ),
          const SizedBox(height: 10),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: color.withOpacity(0.1),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Text(
              "${isUp ? '+' : ''}${fund.gszzl}%",
              style: TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.bold,
                color: color,
              ),
            ),
          ),
          const SizedBox(height: 20),
          Text(
            "更新时间: ${fund.gzTime}",
            style: TextStyle(color: Colors.grey[400], fontSize: 12),
          ),
        ],
      ),
    );
  }
}