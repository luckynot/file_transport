package main

import (
	"log"
	"songxh/file_transport/client"
	"strconv"

	"github.com/andlabs/ui"
)

func main() {
	err := ui.Main(func() {
		var window = ui.NewWindow("文件上传", 800, 100, true)
		serverlabel := ui.NewLabel("服务端地址:")
		serverinput := ui.NewEntry()
		serverinput.SetText("127.0.0.1:10000")
		input := ui.NewEntry()
		input.SetReadOnly(true)
		open := ui.NewButton("打开文件")
		upload := ui.NewButton("上传")
		// 默认未选择文件时无法上传
		upload.Disable()
		//------水平排列的容器
		box1 := ui.NewHorizontalBox()
		box2 := ui.NewHorizontalBox()
		box3 := ui.NewHorizontalBox()
		box1.Append(serverlabel, false)
		box1.Append(serverinput, true)
		box2.Append(input, true)
		box3.Append(open, true)
		box3.Append(upload, true)
		//------垂直排列的容器---------
		div := ui.NewVerticalBox()
		div.Append(box1, true)
		div.Append(box2, true)
		div.Append(box3, true)

		window.SetChild(div)
		// 关闭窗口时退出
		window.OnClosing(func(*ui.Window) bool {
			ui.Quit()
			return true
		})
		// 打开文件按钮点击功能
		open.OnClicked(func(*ui.Button) {
			fn := ui.OpenFile(window)
			if fn == "" {
				return
			}
			input.SetText(fn)
			upload.Enable()
		})
		// 开始上传文件
		upload.OnClicked(func(*ui.Button) {
			upload.Disable()
			defer upload.Enable()
			if input.Text() == "" {
				return
			}
			prochan := make(chan int)
			// 进度条
			progressbar := ui.NewProgressBar()
			fnLabel := ui.NewLabel(input.Text() + ":")
			statLabel := ui.NewLabel("上传中")
			box := ui.NewHorizontalBox()
			box.Append(fnLabel, true)
			box.Append(progressbar, true)
			box.Append(statLabel, true)
			div.Append(box, true)
			go uploadProgress(prochan, progressbar, statLabel)
			client.ServerConn = serverinput.Text()
			go client.Upload(input.Text(), prochan)
		})
		window.Show()
	})
	if err != nil {
		panic(err)
	}
}

// uploadProgress 更新上传进度
func uploadProgress(prochan chan int, progressbar *ui.ProgressBar, statLabel *ui.Label) {
	defer close(prochan)
	var progress int
	for {
		progress = <-prochan
		if progress < 0 {
			switch progress {
			case client.FileInfoErr:
				statLabel.SetText("文件信息错误")
				statLabel.Show()
				break
			case client.ServerConErr:
				statLabel.SetText("服务端连接异常")
				statLabel.Show()
				break
			case client.LoginErr:
				statLabel.SetText("登陆错误")
				statLabel.Show()
				break
			case client.SplitErr:
				statLabel.SetText("拆分方案错误")
				statLabel.Show()
				break
			}
			break
		}
		if progress > 100 {
			progress = 100
		}
		log.Printf("progress:%d\n", progress)
		progressbar.SetValue(progress)
		statLabel.SetText(strconv.Itoa(progress) + "%")
		statLabel.Show()
		progressbar.Show()
		// 上传成功
		if progress == 100 {
			break
		}
	}
}
