v2.7 更新说明

1. main.go、go.mod、app_icon.png：上传到仓库根目录覆盖同名文件。
2. build-macos.yml：进入 .github/workflows/ 后上传，覆盖现有 build-macos.yml。
3. 不要把 vault.dat 上传到公开 GitHub。
4. v2.7 第一次启动时，如果 .app/Contents/Resources/vault.dat 不存在，
   会自动把 ~/Library/Application Support/PasswordBox/vault.dat 迁移进 .app。
   如果两个位置都没有，则首次设置主密码，并在 .app/Contents/Resources/ 新建 V2 vault.dat。
5. V2 加密格式：Argon2id(time=3,memory=65536,threads=4,key=32) + AES-256-GCM。
6. GitHub Actions 会把 app_icon.png 转成 AppIcon.icns，并通过 Info.plist 的 CFBundleIconFile
   让 Finder、桌面、Dock 显示真正的“密码箱.app”图标。
