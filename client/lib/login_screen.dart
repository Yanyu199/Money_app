import 'package:flutter/material.dart';
import 'api/http_service.dart';
import 'main.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _isLoading = false;

  Future<void> _login() async {
    setState(() => _isLoading = true);
    final success = await HttpService.login(
      _usernameController.text,
      _passwordController.text,
    );
    setState(() => _isLoading = false);

    if (success && mounted) {
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => const MainScreen()),
      );
    } else if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text("登录失败，请检查账号密码或服务器地址")),
      );
    }
  }

  Future<void> _register() async {
    setState(() => _isLoading = true);
    final success = await HttpService.register(
      _usernameController.text,
      _passwordController.text,
    );
    setState(() => _isLoading = false);

    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(success ? "注册成功，请登录" : "注册失败")),
      );
    }
  }

  // ✅ 新增：显示服务器配置弹窗
  void _showServerConfigDialog() {
    final urlController = TextEditingController(text: HttpService.baseUrl);
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text("配置服务器地址"),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text("输入 cpolar 新地址 (例如 https://xxx.cpolar.top)", style: TextStyle(fontSize: 12, color: Colors.grey)),
            const SizedBox(height: 10),
            TextField(
              controller: urlController,
              decoration: const InputDecoration(
                border: OutlineInputBorder(),
                hintText: "https://...",
                prefixIcon: Icon(Icons.link),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text("取消")),
          ElevatedButton(
            onPressed: () async {
              // 保存新地址
              await HttpService.setBaseUrl(urlController.text.trim());
              if (mounted) {
                Navigator.pop(ctx);
                ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text("地址已更新，请尝试登录")));
              }
            },
            child: const Text("保存"),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text("盈盈基金"),
        // ✅ 这里的齿轮图标就是入口
        actions: [
          IconButton(
            icon: const Icon(Icons.settings),
            onPressed: _showServerConfigDialog,
            tooltip: "设置服务器地址",
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.trending_up, size: 80, color: Colors.red),
            const SizedBox(height: 20),
            TextField(
              controller: _usernameController,
              decoration: const InputDecoration(labelText: "用户名", prefixIcon: Icon(Icons.person), border: OutlineInputBorder()),
            ),
            const SizedBox(height: 20),
            TextField(
              controller: _passwordController,
              obscureText: true,
              decoration: const InputDecoration(labelText: "密码", prefixIcon: Icon(Icons.lock), border: OutlineInputBorder()),
            ),
            const SizedBox(height: 30),
            _isLoading
                ? const CircularProgressIndicator()
                : Column(
                    children: [
                      SizedBox(
                        width: double.infinity,
                        height: 50,
                        child: ElevatedButton(
                          onPressed: _login,
                          style: ElevatedButton.styleFrom(backgroundColor: Colors.red, foregroundColor: Colors.white),
                          child: const Text("登录", style: TextStyle(fontSize: 18)),
                        ),
                      ),
                      TextButton(
                        onPressed: _register,
                        child: const Text("没有账号？去注册"),
                      ),
                    ],
                  ),
            // 显示当前连接的地址，方便检查
            const SizedBox(height: 20),
            Text("当前连接: ${HttpService.baseUrl}", style: TextStyle(color: Colors.grey[400], fontSize: 10)),
          ],
        ),
      ),
    );
  }
}