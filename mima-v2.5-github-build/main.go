package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

// ==================== 数据模型 ====================
type PasswordEntry struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var allData = []PasswordEntry{}
var filteredData = []PasswordEntry{}
var showPassState = make(map[string]bool)
var currentMasterPassword string

var dataFile string

//go:embed vault_bg.png
var vaultBackground []byte

// vaultDataPath 把数据文件固定放到用户可写的 Application Support 目录。
// Finder 启动 .app 时当前工作目录并不可靠，不能直接写 ./vault.dat。
func vaultDataPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "PasswordBox")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "vault.dat"), nil
}

// migrateLegacyVault 兼容旧版本：如果新位置没有数据，尝试迁移旧的 ./vault.dat。
func migrateLegacyVault(dst string) {
	if _, err := os.Stat(dst); err == nil {
		return
	}
	candidates := []string{"vault.dat"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "vault.dat"))
	}
	for _, src := range candidates {
		if absSrc, _ := filepath.Abs(src); absSrc == dst {
			continue
		}
		b, err := os.ReadFile(src)
		if err == nil && len(b) > 0 {
			_ = os.WriteFile(dst, b, 0600)
			return
		}
	}
}

// 酷炫星云背景：资源直接嵌入程序，打包后无需额外图片文件。
func newVaultBackground() *canvas.Image {
	res := fyne.NewStaticResource("vault_bg.png", vaultBackground)
	img := canvas.NewImageFromResource(res)
	img.FillMode = canvas.ImageFillStretch
	return img
}

// 密码输入前将 macOS 输入源切换为 ABC 英文键盘。
// 非 macOS 平台会自动退化为 no-op。
func focusPasswordEnglish(win fyne.Window, entry fyne.Focusable) {
	switchToEnglishInput()
	win.Canvas().Focus(entry)
}

// EnglishPasswordEntry 每次获得焦点时自动切换到 macOS ABC 英文输入源。
// 这样不仅登录主密码有效，新增/编辑记录里的密码框也同样有效。
type EnglishPasswordEntry struct {
	widget.Entry
}

func newEnglishPasswordEntry() *EnglishPasswordEntry {
	e := &EnglishPasswordEntry{}
	e.Password = true
	e.ExtendBaseWidget(e)
	return e
}

func (e *EnglishPasswordEntry) FocusGained() {
	// 先切一次，再让控件接收焦点，再切一次，避免输入法在焦点切换阶段抢回中文。
	switchToEnglishInput()
	e.Entry.FocusGained()
	switchToEnglishInput()
}

// TypedRune 只接受 ASCII 可打印字符。即使系统输入法临时没有切换成功，
// 密码框也绝不会把中文/全角字符写入主密码。
func (e *EnglishPasswordEntry) TypedRune(r rune) {
	if r >= 0x20 && r <= 0x7e {
		e.Entry.TypedRune(r)
	}
}

// 构建登录/初始化页面中央的暗色玻璃面板。
func vaultGlassPanel(content fyne.CanvasObject, size fyne.Size) fyne.CanvasObject {
	glowOuter := canvas.NewRectangle(color.NRGBA{R: 83, G: 116, B: 255, A: 34})
	glowOuter.SetMinSize(fyne.NewSize(size.Width+28, size.Height+28))
	glowMid := canvas.NewRectangle(color.NRGBA{R: 188, G: 92, B: 255, A: 28})
	glowMid.SetMinSize(fyne.NewSize(size.Width+14, size.Height+14))
	panel := canvas.NewRectangle(color.NRGBA{R: 5, G: 12, B: 34, A: 190})
	panel.SetMinSize(size)
	return container.NewCenter(container.NewStack(glowOuter, glowMid, panel, container.NewPadded(content)))
}

// ==================== 军工级加密引擎 (AES-256-GCM) ====================
func deriveKey(password string) []byte {
	key := []byte(password + "ironman_salt_2026_super_secure")
	for i := 0; i < 10000; i++ {
		hash := sha256.Sum256(key)
		key = hash[:]
	}
	return key
}

func loadAndDecrypt(password string) error {
	ciphertext, err := os.ReadFile(dataFile)
	if err != nil {
		return err
	}
	key := deriveKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return errors.New("数据文件已损坏")
	}
	nonce, cipherTextPart := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherTextPart, nil)
	if err != nil {
		return errors.New("主密码错误，解密失败！")
	}
	return json.Unmarshal(plaintext, &allData)
}

func encryptAndSave(password string) error {
	plaintext, _ := json.Marshal(allData)
	key := deriveKey(password)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return os.WriteFile(dataFile, ciphertext, 0600)
}

// ==================== 国风武将专属配色 ====================
var ColorRed = color.NRGBA{R: 103, G: 117, B: 255, A: 255}
var ColorTeal = color.NRGBA{R: 16, G: 24, B: 58, A: 168}
var ColorGold = color.NRGBA{R: 73, G: 103, B: 205, A: 120}
var ColorArmor = color.NRGBA{R: 5, G: 10, B: 28, A: 112}
var ColorGlass = color.NRGBA{R: 8, G: 16, B: 44, A: 150}

// ==================== 高颜值专属错误弹窗 ====================
func showBeautifulError(win fyne.Window, title, msg string, onClose func()) {
	var popup *widget.PopUp
	bg := canvas.NewRectangle(color.NRGBA{R: 50, G: 30, B: 30, A: 255})
	titleText := canvas.NewText("⚠️ "+title, color.NRGBA{R: 255, G: 100, B: 100, A: 255})
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleText.TextSize = 18

	msgText := canvas.NewText(msg, color.White)
	msgText.Alignment = fyne.TextAlignCenter

	btn := widget.NewButton("我知道了", func() {
		popup.Hide()
		if onClose != nil {
			onClose()
		}
	})
	btn.Importance = widget.DangerImportance

	content := container.NewVBox(
		container.NewCenter(titleText),
		widget.NewLabel(""),
		container.NewCenter(msgText),
		widget.NewLabel(""),
		container.NewCenter(btn),
	)

	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(300, 160))

	popupContent := container.NewStack(bg, spacer, container.NewPadded(content))
	popup = widget.NewModalPopUp(popupContent, win.Canvas())
	popup.Show()
	win.Canvas().Focus(btn)
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("密码箱")
	myWindow.Resize(fyne.NewSize(1100, 700))
	myApp.Settings().SetTheme(&customTheme{Base: myApp.Settings().Theme()})

	var pathErr error
	dataFile, pathErr = vaultDataPath()
	if pathErr != nil {
		showBeautifulError(myWindow, "启动失败", "无法创建本地数据目录："+pathErr.Error(), func() { myApp.Quit() })
		myWindow.ShowAndRun()
		return
	}
	migrateLegacyVault(dataFile)

	var showInitSetup func()
	var showLogin func()
	var showMain func()

	// ==================== 1. 初次使用：设置密码 ====================
	showInitSetup = func() {
		pwd1 := newEnglishPasswordEntry()
		pwd1.SetPlaceHolder("设置主密码 (请务必牢记)")
		pwd2 := newEnglishPasswordEntry()
		pwd2.SetPlaceHolder("再次确认密码")

		doSetup := func() {
			if pwd1.Text == "" || pwd1.Text != pwd2.Text {
				showBeautifulError(myWindow, "验证失败", "两次输入的密码不一致或为空！", func() {
					myWindow.Canvas().Focus(pwd1)
				})
				return
			}
			currentMasterPassword = pwd1.Text
			allData = []PasswordEntry{}
			err := encryptAndSave(currentMasterPassword)
			if err != nil {
				showBeautifulError(myWindow, "致命错误", "创建本地数据文件失败："+err.Error()+"\n位置："+dataFile, nil)
				return
			}
			dialog.ShowInformation("成功", "金库创建成功！\n数据已安全保存在：\n"+dataFile, myWindow)
			showMain()
		}

		pwd1.OnSubmitted = func(s string) { focusPasswordEnglish(myWindow, pwd2) }
		pwd2.OnSubmitted = func(s string) { doSetup() }

		btn := widget.NewButtonWithIcon("创建加密金库", theme.DocumentSaveIcon(), doSetup)
		btn.Importance = widget.HighImportance

		widthSpacer := canvas.NewRectangle(color.Transparent)
		widthSpacer.SetMinSize(fyne.NewSize(350, 0))

		title := canvas.NewText("✦ PASSWORD BOX · FIRST SETUP ✦", color.White)
		title.TextSize = 22
		title.TextStyle = fyne.TextStyle{Bold: true}
		subtitle := canvas.NewText("AES-256-GCM 本地加密 · 主密码不会离开本机", color.NRGBA{R: 207, G: 174, B: 255, A: 255})
		subtitle.Alignment = fyne.TextAlignCenter

		form := container.NewVBox(
			container.NewCenter(title),
			container.NewCenter(subtitle),
			widget.NewLabel(""),
			widthSpacer,
			pwd1, pwd2, btn,
		)
		bg := newVaultBackground()
		shade := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 8, A: 48})
		panel := vaultGlassPanel(form, fyne.NewSize(470, 285))
		myWindow.SetContent(container.NewStack(bg, shade, panel))
		focusPasswordEnglish(myWindow, pwd1)
	}

	// ==================== 2. 日常使用：验证密码 ====================
	showLogin = func() {
		pwdEntry := newEnglishPasswordEntry()
		pwdEntry.SetPlaceHolder("请输入主密码解密数据")

		doLogin := func() {
			err := loadAndDecrypt(pwdEntry.Text)
			if err != nil {
				pwdEntry.SetText("")
				showBeautifulError(myWindow, "解开封印失败", "主密码错误，请重新输入！", func() {
					focusPasswordEnglish(myWindow, pwdEntry)
				})
			} else {
				currentMasterPassword = pwdEntry.Text
				showMain()
			}
		}

		pwdEntry.OnSubmitted = func(s string) { doLogin() }

		loginBtn := widget.NewButtonWithIcon("解开封印", theme.LoginIcon(), doLogin)
		loginBtn.Importance = widget.HighImportance

		widthSpacer := canvas.NewRectangle(color.Transparent)
		widthSpacer.SetMinSize(fyne.NewSize(300, 0))

		title := canvas.NewText("✦ PASSWORD BOX · LOCKED ✦", color.White)
		title.TextSize = 23
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Alignment = fyne.TextAlignCenter
		subtitle := canvas.NewText("STARLIGHT VAULT  ·  LOCAL ENCRYPTION", color.NRGBA{R: 209, G: 176, B: 255, A: 255})
		subtitle.TextSize = 11
		subtitle.Alignment = fyne.TextAlignCenter

		content := container.NewVBox(
			container.NewCenter(title),
			container.NewCenter(subtitle),
			widget.NewLabel(""),
			widthSpacer,
			pwdEntry,
			loginBtn,
		)
		bg := newVaultBackground()
		shade := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 8, A: 42})
		panel := vaultGlassPanel(content, fyne.NewSize(455, 235))
		myWindow.SetContent(container.NewStack(bg, shade, panel))
		focusPasswordEnglish(myWindow, pwdEntry)
	}

	// ==================== 3. 主界面 ====================
	showMain = func() {
		searchEntry := widget.NewEntry()
		searchEntry.SetPlaceHolder("🔍 输入关键字搜索...")
		var table *widget.Table

		filterData := func(keyword string) {
			keyword = strings.ToLower(strings.TrimSpace(keyword))
			if keyword == "" {
				filteredData = append([]PasswordEntry{}, allData...)
			} else {
				filteredData = []PasswordEntry{}
				for _, entry := range allData {
					if strings.Contains(strings.ToLower(entry.Title), keyword) ||
						strings.Contains(strings.ToLower(entry.URL), keyword) ||
						strings.Contains(strings.ToLower(entry.Username), keyword) {
						filteredData = append(filteredData, entry)
					}
				}
			}
			if table != nil {
				table.Refresh()
			}
		}
		searchEntry.OnChanged = filterData

		// ------ 粉色添加与编辑弹窗 ------
		showEditDialog := func(index int, isAdd bool) {
			tEntry, uEntry, nEntry, pEntry := widget.NewEntry(), widget.NewEntry(), widget.NewEntry(), newEnglishPasswordEntry()
			titleStr := "✦ 添加新记录"
			if !isAdd {
				titleStr = "✦ 编辑记录"
				tEntry.SetText(filteredData[index].Title)
				uEntry.SetText(filteredData[index].URL)
				nEntry.SetText(filteredData[index].Username)
				pEntry.SetText(filteredData[index].Password)
			}

			var popup *widget.PopUp

			saveBtn := widget.NewButtonWithIcon("安全保存", theme.DocumentSaveIcon(), func() {
				newEntry := PasswordEntry{Title: tEntry.Text, URL: uEntry.Text, Username: nEntry.Text, Password: pEntry.Text}
				if isAdd {
					allData = append(allData, newEntry)
				} else {
					old := filteredData[index]
					for i, v := range allData {
						if v.Title == old.Title && v.Username == old.Username {
							allData[i] = newEntry
							break
						}
					}
				}
				encryptAndSave(currentMasterPassword)
				filterData(searchEntry.Text)
				popup.Hide()
			})
			saveBtn.Importance = widget.HighImportance

			cancelBtn := widget.NewButtonWithIcon("取消", theme.CancelIcon(), func() { popup.Hide() })

			titleLbl := canvas.NewText(titleStr, color.White)
			titleLbl.TextSize = 18
			titleLbl.TextStyle = fyne.TextStyle{Bold: true}

			makeRow := func(label string, comp fyne.CanvasObject) *fyne.Container {
				l := canvas.NewText(label, color.NRGBA{R: 205, G: 216, B: 255, A: 255})
				l.TextStyle = fyne.TextStyle{Bold: true}
				return container.NewBorder(nil, nil, container.NewPadded(l), nil, comp)
			}

			form := container.NewVBox(
				makeRow("标    题：", tEntry),
				makeRow("网    址：", uEntry),
				makeRow("账    号：", nEntry),
				makeRow("密    码：", pEntry),
			)

			btnBox := container.NewHBox(layout.NewSpacer(), cancelBtn, saveBtn, layout.NewSpacer())

			content := container.NewBorder(
				container.NewPadded(container.NewCenter(titleLbl)),
				container.NewPadded(btnBox),
				nil, nil,
				container.NewPadded(form),
			)

			pinkBg := canvas.NewRectangle(color.NRGBA{R: 8, G: 16, B: 44, A: 238})
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(500, 320))

			popupContent := container.NewStack(pinkBg, spacer, content)
			popup = widget.NewModalPopUp(popupContent, myWindow.Canvas())
			popup.Show()
		}

		searchBtn := widget.NewButtonWithIcon("", theme.SearchIcon(), func() { filterData(searchEntry.Text) })
		addBtn := widget.NewButtonWithIcon("添加记录", theme.ContentAddIcon(), func() { showEditDialog(0, true) })
		addBtn.Importance = widget.HighImportance

		topBar := container.NewBorder(nil, nil, nil, container.NewHBox(searchBtn, addBtn), searchEntry)
		topGlass := canvas.NewRectangle(color.NRGBA{R: 7, G: 15, B: 42, A: 168})
		topContainer := container.NewStack(topGlass, container.NewPadded(topBar))

		table = widget.NewTable(
			func() (int, int) { return len(filteredData) + 1, 5 },
			func() fyne.CanvasObject {
				bg := canvas.NewRectangle(ColorArmor)
				headerTxt := canvas.NewText("", color.Black)
				headerTxt.TextStyle = fyne.TextStyle{Bold: true}
				headerTxt.Alignment = fyne.TextAlignCenter
				dataLbl := widget.NewLabel("")
				dataLbl.Truncation = fyne.TextTruncateEllipsis
				btn1 := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), nil)
				btn1.Importance = widget.LowImportance
				btn2 := widget.NewButtonWithIcon("", theme.VisibilityIcon(), nil)
				btn2.Importance = widget.LowImportance
				btnBox := container.NewHBox(btn2, btn1)

				// 构建边框布局
				cellLayout := container.NewBorder(nil, nil, nil, btnBox, dataLbl)
				return container.NewStack(bg, headerTxt, cellLayout)
			},
			func(id widget.TableCellID, cell fyne.CanvasObject) {
				stack := cell.(*fyne.Container)
				bg := stack.Objects[0].(*canvas.Rectangle)
				headerTxt := stack.Objects[1].(*canvas.Text)
				cellLayout := stack.Objects[2].(*fyne.Container)

				// 【核心防崩溃修复】：动态安全类型断言，彻底防止底层打乱顺序导致的崩溃
				var dataLbl *widget.Label
				var btnBox *fyne.Container

				for _, obj := range cellLayout.Objects {
					if l, ok := obj.(*widget.Label); ok {
						dataLbl = l
					} else if c, ok := obj.(*fyne.Container); ok {
						btnBox = c
					}
				}

				btn2 := btnBox.Objects[0].(*widget.Button)
				btn1 := btnBox.Objects[1].(*widget.Button)

				row, col := id.Row, id.Col
				headerTxt.Hide()
				cellLayout.Hide()
				btnBox.Hide()
				btn2.Show()

				if row == 0 {
					bg.FillColor = color.NRGBA{R: 40, G: 62, B: 130, A: 105}
					headers := []string{"标题", "网址", "用户名", "密码", "操作"}
					headerTxt.Text = headers[col]
					headerTxt.Show()
				} else {
					bg.FillColor = color.NRGBA{R: 5, G: 13, B: 38, A: 108}
					cellLayout.Show()
					if row-1 >= len(filteredData) {
						return
					}
					entry := filteredData[row-1]
					uid := entry.Title + entry.Username

					switch col {
					case 0:
						dataLbl.SetText(entry.Title)
					case 1, 2:
						val := entry.URL
						if col == 2 {
							val = entry.Username
						}
						dataLbl.SetText(val)
						btn1.SetIcon(theme.ContentCopyIcon())
						btn1.OnTapped = func() { myWindow.Clipboard().SetContent(val) }
						btn2.Hide()
						btnBox.Show()
					case 3:
						if showPassState[uid] {
							dataLbl.SetText(entry.Password)
							btn2.SetIcon(theme.VisibilityOffIcon())
						} else {
							dataLbl.SetText("••••••••")
							btn2.SetIcon(theme.VisibilityIcon())
						}
						btn1.SetIcon(theme.ContentCopyIcon())
						btn2.OnTapped = func() { showPassState[uid] = !showPassState[uid]; table.Refresh() }
						btn1.OnTapped = func() { myWindow.Clipboard().SetContent(entry.Password) }
						btnBox.Show()
					case 4:
						dataLbl.SetText("")
						btn2.SetIcon(theme.DocumentCreateIcon())
						btn1.SetIcon(theme.DeleteIcon())
						btn2.OnTapped = func() { showEditDialog(row-1, false) }
						btn1.OnTapped = func() {
							dialog.ShowConfirm("删除确认", "确定删除 ["+entry.Title+"] 吗？", func(b bool) {
								if b {
									for i, v := range allData {
										if v.Title == entry.Title && v.Username == entry.Username {
											allData = append(allData[:i], allData[i+1:]...)
											break
										}
									}
									encryptAndSave(currentMasterPassword)
									filterData(searchEntry.Text)
								}
							}, myWindow)
						}
						btnBox.Show()
					}
				}
				bg.Refresh()
				headerTxt.Refresh()
			},
		)

		table.SetColumnWidth(0, 120)
		table.SetColumnWidth(1, 220)
		table.SetColumnWidth(2, 160)
		table.SetColumnWidth(3, 240)
		table.SetColumnWidth(4, 100)

		filterData("")
		mainBg := newVaultBackground()
		mainShade := canvas.NewRectangle(color.NRGBA{R: 0, G: 4, B: 18, A: 62})
		mainGlass := canvas.NewRectangle(color.NRGBA{R: 4, G: 10, B: 30, A: 76})
		mainUI := container.NewBorder(topContainer, nil, nil, nil, table)
		myWindow.SetContent(container.NewStack(mainBg, mainShade, mainGlass, mainUI))
	}

	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		showInitSetup()
	} else {
		showLogin()
	}

	myWindow.ShowAndRun()
}

// ==================== 主题引擎 ====================
type customTheme struct{ Base fyne.Theme }

func (t *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return ColorArmor
	case theme.ColorNamePrimary:
		return ColorRed
	case theme.ColorNameButton:
		return ColorTeal
	case theme.ColorNameInputBackground:
		return color.NRGBA{8, 17, 45, 190}
	case theme.ColorNameForeground:
		return color.White
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{150, 150, 150, 255}
	case theme.ColorNameHover:
		return color.NRGBA{70, 90, 160, 110}
	case theme.ColorNameSeparator:
		return color.Transparent
	}
	return t.Base.Color(name, theme.VariantDark)
}

func (t *customTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameSeparatorThickness {
		return 0
	}
	return t.Base.Size(name)
}

func (t *customTheme) Font(style fyne.TextStyle) fyne.Resource    { return t.Base.Font(style) }
func (t *customTheme) Icon(name fyne.ThemeIconName) fyne.Resource { return t.Base.Icon(name) }
