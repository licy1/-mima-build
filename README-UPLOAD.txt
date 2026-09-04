MIMA 密码箱 v2.6 上传说明

建议：把本包解压后，将 main.go、vault_bg.png、go.mod、go.sum、inputsource_darwin.go 上传到仓库根目录覆盖旧文件。
然后把 .github/workflows/build-macos.yml 的内容覆盖仓库中的同名 workflow。

本版识别特征：窗口标题必须显示“密码箱 v2.6”；背景不包含任何登录框、MIMA 字样、表格或按钮；这些 UI 都由 Go/Fyne 代码实时绘制。

数据路径保持：~/Library/Application Support/PasswordBox/vault.dat
加密格式和 v2.5 保持兼容。
