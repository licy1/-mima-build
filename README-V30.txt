MIMA 密码箱 v3.0 更新

1. 修复 macOS 26 App Translocation/只读目录问题。
2. /private/var/folders 下启动时强制安装到 ~/Applications/密码箱.app。
3. 安装失败时停止继续写 vault.dat，并显示明确错误。
4. vault.dat 仍保存在 密码箱.app/Contents/Resources/vault.dat。
5. 保持 V2 Argon2id + AES-256-GCM 加密。
6. 保持科技风 AppIcon。

上传：main.go、build-macos.yml；app_icon.png 可一并覆盖。
不要上传 vault.dat。
