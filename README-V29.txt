MIMA 密码箱 v2.9 更新：
1. macOS 26 不再依赖 AppTranslocation 路径名，直接检测 Resources 是否可写。
2. 不可写时自动安装到 ~/Applications/密码箱.app 并重新启动。
3. vault.dat 继续保存在 .app/Contents/Resources/；没有时创建。
4. 保留 V2 Argon2id + AES-256-GCM。
5. 更换科技风保险库 App 图标。
6. 错误信息自动换行。
