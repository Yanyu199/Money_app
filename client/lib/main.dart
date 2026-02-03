import 'package:flutter/material.dart';
import 'dart:async';
import 'api/http_service.dart';
import 'models/fund_model.dart';
import 'login_screen.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await HttpService.initBaseUrl();
  final isLoggedIn = await HttpService.tryAutoLogin();
  runApp(MyApp(initialRoute: isLoggedIn ? const MainScreen() : const LoginScreen()));
}

class MyApp extends StatelessWidget {
  final Widget initialRoute;
  const MyApp({super.key, required this.initialRoute});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.red, brightness: Brightness.light),
        useMaterial3: true,
        scaffoldBackgroundColor: const Color(0xFFF5F5F5),
        appBarTheme: const AppBarTheme(backgroundColor: Colors.red, foregroundColor: Colors.white, centerTitle: true),
      ),
      home: initialRoute,
    );
  }
}

// 🔥 排序类型枚举
enum SortType { rate, profit }

class MainScreen extends StatefulWidget {
  const MainScreen({super.key});
  @override
  State<MainScreen> createState() => _MainScreenState();
}

class _MainScreenState extends State<MainScreen> {
  int _currentIndex = 0;
  final Map<String, FundModel> _marketData = {}; 
  Map<String, double> _myHoldingsConfig = {};    
  List<String> _myWatchlistConfig = [];          
  
  bool _isDescSort = true; // 降序/升序
  SortType _sortType = SortType.rate; // 🔥 排序模式：按涨幅还是按金额
  
  bool _isLoadingData = true;
  bool _isPrivacyMode = false;
  String _updateTimeStr = "--:--:--"; 

  @override
  void initState() {
    super.initState();
    _initData();
  }

  Future<void> _initData() async {
    await _loadMyConfig();
    await _fetchLatestMarket();
  }

  Future<void> _loadMyConfig() async {
    final data = await HttpService.getMyData();
    if (data != null && mounted) {
      final holdings = data['holdings'] as List;
      final watchlist = data['watchlist'] as List;
      setState(() {
        _myHoldingsConfig = { for (var h in holdings) h['fund_code']: (h['amount'] as num).toDouble() };
        _myWatchlistConfig = watchlist.map<String>((w) => w['fund_code'].toString()).toList();
        _isLoadingData = false;
      });
    }
  }

  Future<void> _fetchLatestMarket() async {
    final list = await HttpService.refreshMarketData();
    if (mounted) {
      setState(() {
        for (var item in list) {
          final fund = FundModel.fromJson(item);
          _marketData[fund.fundCode] = fund;
        }
        final now = DateTime.now();
        _updateTimeStr = "${now.hour.toString().padLeft(2, '0')}:${now.minute.toString().padLeft(2, '0')}:${now.second.toString().padLeft(2, '0')}";
      });
    }
  }

  Future<void> _onRefresh() async {
    await Future.wait([_loadMyConfig(), _fetchLatestMarket()]);
  }

  void _logout() {
    HttpService.logout();
    Navigator.of(context).pushReplacement(MaterialPageRoute(builder: (_) => const LoginScreen()));
  }

  Future<void> _handleDelete(String code, String type) async {
    setState(() {
      if (type == 'holding') _myHoldingsConfig.remove(code);
      else _myWatchlistConfig.remove(code);
    });
    await HttpService.deleteFund(code, type);
  }

  void _showSettleDialog() {
    showDialog(context: context, builder: (ctx) => AlertDialog(
      title: const Text("确认更新持仓?"),
      content: const Text("系统将根据【最新的官方净值/收盘价】自动更新你所有持仓的金额。\n\n⚠️ 建议在每晚 22:00 后或次日早上操作。"),
      actions: [
        TextButton(onPressed: () => Navigator.pop(ctx), child: const Text("取消")),
        ElevatedButton(
          style: ElevatedButton.styleFrom(backgroundColor: Colors.red, foregroundColor: Colors.white),
          onPressed: () async {
            Navigator.pop(ctx);
            ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text("正在结算中...")));
            String msg = await HttpService.settleHoldings();
            if (mounted) {
               showDialog(context: context, builder: (_) => AlertDialog(title: const Text("结算完成"), content: Text(msg), actions: [TextButton(onPressed: ()=>Navigator.pop(context), child: const Text("好"))]));
               _onRefresh(); 
            }
          },
          child: const Text("确定结算"),
        )
      ],
    ));
  }

  // 🔥 排序弹窗
  void _showSortDialog() {
    showModalBottomSheet(context: context, builder: (ctx) => Column(mainAxisSize: MainAxisSize.min, children: [
      const Padding(padding: EdgeInsets.all(16), child: Text("排序方式", style: TextStyle(fontWeight: FontWeight.bold))),
      ListTile(
        leading: const Icon(Icons.show_chart),
        title: const Text("按涨跌幅排序"),
        trailing: _sortType == SortType.rate ? Icon(_isDescSort ? Icons.arrow_downward : Icons.arrow_upward, color: Colors.red) : null,
        onTap: () {
          setState(() {
            if (_sortType == SortType.rate) _isDescSort = !_isDescSort;
            else { _sortType = SortType.rate; _isDescSort = true; }
          });
          Navigator.pop(ctx);
        },
      ),
      ListTile(
        leading: const Icon(Icons.attach_money),
        title: const Text("按当日收益额排序"),
        trailing: _sortType == SortType.profit ? Icon(_isDescSort ? Icons.arrow_downward : Icons.arrow_upward, color: Colors.red) : null,
        onTap: () {
          setState(() {
            if (_sortType == SortType.profit) _isDescSort = !_isDescSort;
            else { _sortType = SortType.profit; _isDescSort = true; }
          });
          Navigator.pop(ctx);
        },
      ),
    ]));
  }

  // 🔥 核心排序逻辑更新
  List<FundModel> _getSortedList(List<String> codes) {
    var list = codes.map((c) => _marketData[c]).whereType<FundModel>().toList();
    list.sort((a, b) {
      double valA = 0, valB = 0;
      double rateA = double.tryParse(a.gszzl) ?? 0.0;
      double rateB = double.tryParse(b.gszzl) ?? 0.0;

      if (_sortType == SortType.rate) {
        valA = rateA;
        valB = rateB;
      } else {
        // 按收益额排序：涨跌幅 * 本金
        double principalA = _myHoldingsConfig[a.fundCode] ?? 0;
        double principalB = _myHoldingsConfig[b.fundCode] ?? 0;
        valA = principalA * rateA;
        valB = principalB * rateB;
      }
      return _isDescSort ? valB.compareTo(valA) : valA.compareTo(valB);
    });
    return list;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(_currentIndex == 0 ? "我的持仓" : "自选关注", style: const TextStyle(fontWeight: FontWeight.bold)),
        leading: IconButton(icon: const Icon(Icons.logout), onPressed: _logout),
        actions: [
          if (_currentIndex == 0) IconButton(icon: const Icon(Icons.check_circle_outline), tooltip: "一键结算", onPressed: _showSettleDialog),
          IconButton(icon: Icon(_isPrivacyMode ? Icons.visibility_off : Icons.visibility), onPressed: () => setState(() => _isPrivacyMode = !_isPrivacyMode)),
          // 🔥 排序按钮改为弹窗
          IconButton(icon: const Icon(Icons.sort), onPressed: _showSortDialog),
          IconButton(icon: const Icon(Icons.add_circle_outline), onPressed: () => _showAddFundDialog(_currentIndex == 0 ? "holding" : "watchlist")),
        ],
      ),
      body: _isLoadingData 
        ? const Center(child: CircularProgressIndicator()) 
        : RefreshIndicator(onRefresh: _onRefresh, color: Colors.red, child: _currentIndex == 0 ? _buildHoldingsTab() : _buildWatchlistTab()),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex,
        onDestinationSelected: (index) => setState(() => _currentIndex = index),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.account_balance_wallet_outlined), selectedIcon: Icon(Icons.account_balance_wallet), label: "持仓"),
          NavigationDestination(icon: Icon(Icons.favorite_border), selectedIcon: Icon(Icons.favorite), label: "自选"),
        ],
      ),
    );
  }

  Widget _buildHoldingsTab() {
    final list = _getSortedList(_myHoldingsConfig.keys.toList());
    double totalProfit = 0;
    double totalPrincipal = 0;

    for (var code in _myHoldingsConfig.keys) {
      final fund = _marketData[code];
      final principal = _myHoldingsConfig[code] ?? 0;
      totalPrincipal += principal;
      if (fund != null) totalProfit += principal * ((double.tryParse(fund.gszzl) ?? 0) / 100);
    }

    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      children: [
        _buildProfitHeader(totalProfit, totalPrincipal, list.length),
        if (list.isEmpty) SizedBox(height: 300, child: _buildEmptyState("暂无持仓，点击右上角+号添加")),
        ...list.map((fund) {
          final principal = _myHoldingsConfig[fund.fundCode] ?? 0;
          final profit = principal * ((double.tryParse(fund.gszzl) ?? 0) / 100);
          return Dismissible(
            key: Key("holding_${fund.fundCode}"),
            direction: DismissDirection.endToStart,
            background: Container(color: Colors.red, alignment: Alignment.centerRight, padding: const EdgeInsets.only(right: 20), child: const Icon(Icons.delete, color: Colors.white)),
            confirmDismiss: (d) async => await showDialog(context: context, builder: (ctx) => AlertDialog(title: const Text("确认删除?"), actions: [TextButton(onPressed: ()=>Navigator.pop(ctx,false),child:const Text("取消")),TextButton(onPressed: ()=>Navigator.pop(ctx,true),child:const Text("删除"))])),
            onDismissed: (d) => _handleDelete(fund.fundCode, "holding"),
            child: GestureDetector(
              onTap: () => _gotoDetail(fund),
              onLongPress: () => _showEditAmountDialog(fund.fundCode, principal),
              child: _buildFundCard(fund, true, profit, principal),
            ),
          );
        }).toList(),
        const SizedBox(height: 20),
      ],
    );
  }

  Widget _buildWatchlistTab() {
    final list = _getSortedList(_myWatchlistConfig);
    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      children: [
        if (list.isEmpty) SizedBox(height: 500, child: _buildEmptyState("暂无自选，点击右上角+号添加")),
        ...list.map((fund) => Dismissible(
          key: Key("watch_${fund.fundCode}"),
          direction: DismissDirection.endToStart,
          background: Container(color: Colors.red, alignment: Alignment.centerRight, padding: const EdgeInsets.only(right: 20), child: const Icon(Icons.delete, color: Colors.white)),
          onDismissed: (d) => _handleDelete(fund.fundCode, "watchlist"),
          child: GestureDetector(onTap: () => _gotoDetail(fund), child: _buildFundCard(fund, false, 0, 0)),
        )).toList()
      ],
    );
  }

  Widget _buildProfitHeader(double profit, double totalPrincipal, int count) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: const BoxDecoration(color: Colors.red, borderRadius: BorderRadius.vertical(bottom: Radius.circular(24))),
      child: Column(children: [
          const Text("当日预估总收益", style: TextStyle(color: Colors.white70)),
          const SizedBox(height: 5),
          Text(_isPrivacyMode ? "****" : "${profit > 0 ? '+' : ''}${profit.toStringAsFixed(2)}", style: const TextStyle(color: Colors.white, fontSize: 36, fontWeight: FontWeight.bold)),
          const SizedBox(height: 10),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
            decoration: BoxDecoration(color: Colors.white.withOpacity(0.2), borderRadius: BorderRadius.circular(12)),
            child: Text("持有本金: ${_isPrivacyMode ? "****" : '¥' + totalPrincipal.toStringAsFixed(0)}", style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 13)),
          ),
          const SizedBox(height: 15),
          Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              _headerInfoItem(Icons.pie_chart, "持有 $count 支"),
              Container(width: 1, height: 12, color: Colors.white38, margin: const EdgeInsets.symmetric(horizontal: 15)),
              _headerInfoItem(Icons.access_time, "更新 $_updateTimeStr"),
          ])
      ]),
    );
  }
  Widget _headerInfoItem(IconData icon, String text) => Row(children: [Icon(icon, color: Colors.white70, size: 14), const SizedBox(width: 4), Text(text, style: const TextStyle(color: Colors.white70, fontSize: 12))]);
  Widget _buildEmptyState(String text) => Center(child: Text(text, style: TextStyle(color: Colors.grey[500])));
  
  Widget _buildTag(String timeStr) {
    String text = "估"; Color color = Colors.orange;
    if (timeStr.contains("实时")) { text = "实"; color = Colors.purple; } 
    else if (timeStr.contains("确")) { text = "确"; color = Colors.grey; }
    return Container(padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2), margin: const EdgeInsets.only(left: 4), decoration: BoxDecoration(color: color.withOpacity(0.1), border: Border.all(color: color, width: 0.5), borderRadius: BorderRadius.circular(4)), child: Text(text, style: TextStyle(fontSize: 10, color: color)));
  }

  Widget _buildFundCard(FundModel fund, bool isHolding, double profit, double principal) {
     final rate = double.tryParse(fund.gszzl) ?? 0.0;
     final color = rate >= 0 ? Colors.red : Colors.green;
     return Card(elevation: 0, color: Colors.white, margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 6), child: Padding(padding: const EdgeInsets.all(16), child: Row(children: [
       Expanded(flex: 4, child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
         Text(fund.name, style: const TextStyle(fontWeight: FontWeight.bold), maxLines: 1), 
         Row(children: [
           Text(fund.fundCode, style: const TextStyle(color: Colors.grey, fontSize: 11)),
           // 🔥 小数点优化: .toStringAsFixed(2)
           if (isHolding) Text(" 持有:${_isPrivacyMode ? '****' : principal.toStringAsFixed(2)}", style: const TextStyle(color: Colors.grey, fontSize: 11)),
           if (fund.premiumRate.isNotEmpty) Container(margin: const EdgeInsets.only(left: 8), padding: const EdgeInsets.symmetric(horizontal: 4), decoration: BoxDecoration(color: Colors.blue[50], borderRadius: BorderRadius.circular(4)), child: Text("溢价${fund.premiumRate}", style: const TextStyle(fontSize: 10, color: Colors.blue)))
         ])
       ])), 
       Column(crossAxisAlignment: CrossAxisAlignment.end, children: [
         Row(children: [
           Text(isHolding && !_isPrivacyMode ? "${profit>0?'+':''}${profit.toStringAsFixed(2)}" : fund.gsz, style: TextStyle(color: isHolding ? color : Colors.black87, fontSize: 18, fontWeight: FontWeight.bold)),
           _buildTag(fund.gzTime)
         ]),
         Text(isHolding ? "当日收益" : "净值/估值", style: const TextStyle(fontSize: 10, color: Colors.grey))
       ]), 
       const SizedBox(width: 12), 
       Container(width: 72, height: 32, alignment: Alignment.center, decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(6)), child: Text("${rate>=0?'+':''}$rate%", style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold)))
     ])));
  }

  void _showAddFundDialog(String targetType) {
    showDialog(context: context, builder: (ctx) => _SearchDialog(targetType: targetType, onAdded: () {
      if (mounted) { Navigator.pop(ctx); ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text("添加成功"))); _onRefresh(); }
    }));
  }

  void _showEditAmountDialog(String code, double current) {
    String input = current.toString();
    showDialog(context: context, builder: (ctx) => AlertDialog(
      title: const Text("修改持仓金额"),
      content: TextField(decoration: InputDecoration(hintText: "$current"), keyboardType: TextInputType.number, onChanged: (v) => input = v),
      actions: [
        ElevatedButton(onPressed: () async {
          await HttpService.addFundDB(code, "holding", double.tryParse(input) ?? 0);
          Navigator.pop(ctx);
          _onRefresh();
        }, child: const Text("保存")),
      ],
    ));
  }

  void _gotoDetail(FundModel fund) => Navigator.push(context, MaterialPageRoute(builder: (_) => DetailScreen(fund: fund)));
}

// 搜索组件保持不变 (为节省空间省略，因为刚才已经给你了，没变)
class _SearchDialog extends StatefulWidget {
  final String targetType;
  final VoidCallback onAdded;
  const _SearchDialog({required this.targetType, required this.onAdded});
  @override
  State<_SearchDialog> createState() => _SearchDialogState();
}
class _SearchDialogState extends State<_SearchDialog> {
  final _searchController = TextEditingController();
  final _amountController = TextEditingController();
  List<dynamic> _searchResults = [];
  bool _isSearching = false;
  dynamic _selectedFund;
  void _doSearch(String key) async {
    if (key.isEmpty) return;
    setState(() => _isSearching = true);
    final results = await HttpService.searchFund(key);
    if (mounted) setState(() { _searchResults = results; _isSearching = false; });
  }
  @override
  Widget build(BuildContext context) {
    bool isHolding = widget.targetType == "holding";
    return AlertDialog(
      title: Text(isHolding ? "添加持仓" : "添加自选"),
      content: SizedBox(width: double.maxFinite, child: Column(mainAxisSize: MainAxisSize.min, children: [
          TextField(controller: _searchController, decoration: InputDecoration(hintText: "输入代码或名称搜索", prefixIcon: const Icon(Icons.search), suffixIcon: IconButton(icon: const Icon(Icons.arrow_forward), onPressed: () => _doSearch(_searchController.text))), onSubmitted: _doSearch),
          const SizedBox(height: 10),
          if (_isSearching) const LinearProgressIndicator(),
          if (_selectedFund == null) SizedBox(height: 200, child: ListView.builder(itemCount: _searchResults.length, itemBuilder: (ctx, i) { final item = _searchResults[i]; return ListTile(title: Text(item['name']), subtitle: Text("${item['code']} - ${item['type']}"), onTap: () => setState(() => _selectedFund = item)); })),
          if (_selectedFund != null) ...[
            Container(padding: const EdgeInsets.all(8), decoration: BoxDecoration(color: Colors.grey[200], borderRadius: BorderRadius.circular(8)), child: Row(children: [Expanded(child: Text("${_selectedFund['name']} (${_selectedFund['code']})", style: const TextStyle(fontWeight: FontWeight.bold))), IconButton(icon: const Icon(Icons.close, size: 16), onPressed: () => setState(() => _selectedFund = null))])),
            if (isHolding) ...[const SizedBox(height: 10), TextField(controller: _amountController, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: "持有金额", prefixIcon: Icon(Icons.attach_money)))]
          ]
      ])),
      actions: [TextButton(onPressed: () => Navigator.pop(context), child: const Text("取消")), ElevatedButton(onPressed: _selectedFund == null ? null : () async { double amt = isHolding ? (double.tryParse(_amountController.text) ?? 0) : 0; await HttpService.addFundDB(_selectedFund['code'], widget.targetType, amt); widget.onAdded(); }, child: const Text("确定"))],
    );
  }
}

// 🔥 全新升级的详情页：展示实时重仓股
class DetailScreen extends StatefulWidget {
  final FundModel fund;
  const DetailScreen({super.key, required this.fund});
  @override
  State<DetailScreen> createState() => _DetailScreenState();
}
class _DetailScreenState extends State<DetailScreen> {
  Map<String, dynamic>? detailData;
  bool _isLoading = true;

  @override
  void initState() { super.initState(); _fetchDetail(); }

  Future<void> _fetchDetail() async {
    final val = await HttpService.getDetail(widget.fund.fundCode);
    if (mounted) setState(() { detailData = val; _isLoading = false; });
  }

  @override
  Widget build(BuildContext context) {
    final stockList = detailData != null ? (detailData!['stock_details'] as List?) : null;

    return Scaffold(
      appBar: AppBar(title: Text("${widget.fund.name} (${widget.fund.fundCode})")),
      body: RefreshIndicator(
        onRefresh: _fetchDetail,
        child: SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          child: Column(children: [
            // 1. 走势图
            Container(color: Colors.white, padding: const EdgeInsets.all(10), child: Image.network("http://j3.dfcfw.com/images/fav/charts/t${widget.fund.fundCode}.png", fit: BoxFit.contain)),
            const SizedBox(height: 10),
            
            // 2. 重仓股列表头
            Container(padding: const EdgeInsets.all(16), alignment: Alignment.centerLeft, child: const Text("重仓持股 (实时行情)", style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16))),
            
            // 3. 股票列表
            if (_isLoading) const Padding(padding: EdgeInsets.all(20), child: CircularProgressIndicator())
            else if (stockList == null || stockList.isEmpty) const Padding(padding: EdgeInsets.all(20), child: Text("暂无持仓数据"))
            else ...stockList.map((s) {
              final changeStr = s['change'] as String;
              final isUp = !changeStr.startsWith("-");
              final color = isUp ? Colors.red : Colors.green;
              return Container(
                color: Colors.white,
                margin: const EdgeInsets.only(bottom: 1),
                child: ListTile(
                  title: Text(s['name'], style: const TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: Text(s['code'], style: const TextStyle(color: Colors.grey, fontSize: 12)),
                  trailing: SizedBox(width: 120, child: Row(mainAxisAlignment: MainAxisAlignment.end, children: [
                    Text(s['price'], style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                    const SizedBox(width: 10),
                    Container(width: 60, padding: const EdgeInsets.symmetric(vertical: 4), alignment: Alignment.center, decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(4)), child: Text(changeStr, style: const TextStyle(color: Colors.white, fontSize: 12)))
                  ])),
                ),
              );
            }).toList(),
            
            const SizedBox(height: 30),
          ]),
        ),
      ),
    );
  }
}