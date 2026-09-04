密码箱 v2.5 GitHub 自动构建包

上传方法：
1. 解压本 ZIP。
2. 把“解压后的所有文件和文件夹”上传到 GitHub 仓库 lic y1/-mima-build 的根目录。
   注意：必须连同隐藏目录 .github 一起上传；不要只上传这个 ZIP 文件。
3. Commit changes 到 main。
4. GitHub Actions 会自动启动 Build macOS ARM64 App。
5. 构建成功后，在 Actions -> 对应运行 -> Artifacts 下载 mima-v2.5-mac-arm64。

重要：不要上传 vault.dat，它包含你的真实密码库数据。
