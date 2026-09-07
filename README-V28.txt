MIMA 密码箱 v2.8 更新

修复：macOS App Translocation 导致 .app/Contents/Resources/vault.dat 只读。
首次从隔离临时路径启动时，程序会自动复制到 ~/Applications/密码箱.app，移除 quarantine，重新 ad-hoc 签名并启动可写版本。
vault.dat 仍保存在 密码箱.app/Contents/Resources/vault.dat；不存在时创建。
升级时若 ~/Applications/密码箱.app 内已有 vault.dat，会先保存并恢复，避免覆盖数据。
继续使用 V2：Argon2id + AES-256-GCM。
