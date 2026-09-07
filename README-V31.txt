密码箱 v3.1 稳定版更新

本版移除 v2.7-v3.0 的 App 自复制、自修改和运行时重签名逻辑。
数据恢复为 v2.4 已验证的稳定位置：~/Library/Application Support/PasswordBox/vault.dat
加密仍为 V2 Argon2id + AES-256-GCM；科技风 UI 和应用图标保留。

上传覆盖：
1. main.go -> 仓库根目录
2. build-macos.yml -> .github/workflows/

无需上传 vault.dat。
