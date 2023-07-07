package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var zhub *Client

var (
	addr = "127.0.0.1:1216"
)

func init() {
	client, err := Create("zhub-cli", addr, "C-0", "admin@123456")
	if err != nil {
		log.Fatal(err)
	}
	zhub = client
}

func newClient(appname, groupid string) *Client {
	client, err := Create(appname, addr, groupid, "admin@123456")
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func TestTimer(t *testing.T) {
	go func() {
		client := newClient("zhub-cli", "g-1")

		client.Subscribe("ax1", func(v string) {
			log.Println("topic-1-ax: " + v)
		})
	}()
	go func() {
		client := newClient("zhub-cli", "g-1")

		client.Subscribe("ax1", func(v string) {
			log.Println("topic-2-ax: " + v)
		})
	}()

	go func() {
		client := newClient("zhub-cli", "g-1")

		client.Subscribe("ax1", func(v string) {
			log.Println("topic-3-ax: " + v)
		})
	}()

	time.Sleep(time.Hour * 3)
}

func TestSendCmd(t *testing.T) {
	client := newClient("zhub-cli", "group-admin")

	//client.Cmd("reload-timer")
	client.Cmd("shutdown")
}

func TestPublish(t *testing.T) {

	/*zhub.Publish("abx", "asd\r\nxxx1")
	zhub.Publish("abx", "asd\r\nxxx2")
	zhub.Publish("abx", "asd\r\nxxx3")
	zhub.Publish("abx", "asd\r\nxxx4")
	zhub.Publish("abx", "asd\r\nxxx5")*/

	/*for i := 0; i < 10000; i++ {
		zhub.Publish("ax1", strconv.Itoa(i))
	}*/

	/*for i := 0; i < 20_0000; i++ {
		time.Sleep(1 * time.Millisecond)
		zhub.Publish("b", strconv.Itoa(i))
	}*/

	/*zhub.Subscribe("wx:user-follow", func(v string) {
		fmt.Println(v)
	})*/

	//hub.Publish("ax", "1")

	time.Sleep(time.Second)
}

func TestLock(t *testing.T) {
	client, _ := Create("zhub-cli", addr, "xx", "admin@123456")

	client.Subscribe("lock", func(v string) {

	})

	var fun = func(x string) {
		log.Println("lock", time.Now().UnixNano()/1e6)
		lock := client.Lock("a", 30)
		defer client.Unlock(lock)
		//client.Lock("a", 5)

		for i := 0; i < 5; i++ {
			time.Sleep(time.Second * 1)
			fmt.Println(x + ":" + strconv.Itoa(i+1))
		}
	}

	go fun("x")
	go fun("y")
	go fun("z")

	time.Sleep(time.Second * 30 * 10)
}

func rotate(nums []int, k int) {
	k = k % len(nums)
	nums = append(nums[len(nums)-k:], nums[0:len(nums)-k]...)
}
func TestName(t *testing.T) {
	//str := ", response = {\"success\":true,\"retcode\":0,\"result\":{\"age\":0,\"explevel\":1,\"face\":\"https://aimg.woaihaoyouxi.com/haogame/202106/pic/20210629095545FmGt-v9NYqyNZ_Q6_y3zM_RMrDgd.jpg\",\"followed\":0,\"gender\":0,\"idenstatus\":0,\"matchcatelist\":[{\"catename\":\"足球\",\"catepic\":\"https://aimg.woaihaoyouxi.com/haogame/202107/pic/20210714103556FoG5ICf_7BFx6Idyo3TYpJQ7tmfG.png\",\"matchcateid\":1},{\"catename\":\"篮球\",\"catepic\":\"https://aimg.woaihaoyouxi.com/haogame/202107/pic/20210714103636FklsXTn1f6Jlsam8Jk-yFB7Upo3C.png\",\"matchcateid\":2}],\"matchcates\":\"2,1\",\"mobile\":\"18515190967\",\"regtime\":1624931714781,\"sessionid\":\"d1fc447753bd4700ad29674a753030fa\",\"status\":10,\"userid\":100463,\"username\":\"绝尘\",\"userno\":100463}}"
	/*str := "别人家的女娃子🤞🏻"
	str = "❦别人家的女娃子🤞🏻ꚢ"

	//str2 := strings.ReplaceAll(str, "^[\\u]$", "*")
	//fmt.Println(str2)

	inString := utf8.RuneCountInString(str)


	fmt.Println(inString)
	fmt.Println(len(str))
	*/

	// 10进制 -> 36进制
	// fmt.Println(strconv.FormatInt(50000, 36))

	nums := []int{1, 2, 3, 4, 5, 6, 7}
	k := 8
	rotate(nums, k)
	fmt.Println(nums)
}

func toStr(d interface{}) string {
	bs, _ := json.Marshal(d)
	return string(bs[:])
}

// 接收数据 A
func TestC_a(t *testing.T) {
	zhub, err := Create("zhub-cli", addr, "C-1", "admin@123456")

	if err != nil {
		log.Fatal(err)
	}

	zhub.Subscribe("cmt:user-msg", func(v string) {
		fmt.Println(v)
	})

	time.Sleep(10 * time.Hour)
}

// 接收数据
func TestC_ab(t *testing.T) {
	zhub, err := Create("zhub-cli", addr, "C-1", "admin@123456")

	if err != nil {
		log.Fatal(err)
	}

	zhub.Subscribe("a", func(v string) {
		fmt.Println("a:", v)
	})
	zhub.Subscribe("b", func(v string) {
		fmt.Println("b：", v)
	})
	zhub.Subscribe("im:friend:186", func(v string) {
		fmt.Println("im:friend:186：", v)
	})

	time.Sleep(1 * time.Hour)
}

func TestDelay2(t *testing.T) {
	zhub, err := Create("zhub-cli", addr, "C-1", "admin@123456")
	if err != nil {
		log.Fatal(err)
	}

	var x int64 = 0
	go func() {
		zhub.Subscribe("a", func(v string) {
			fmt.Println(v, "-", atomic.AddInt64(&x, 1))
		})
	}()

	zhub2, err := Create("zhub-cli", addr, "C-1", "admin@123456")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		zhub2.Subscribe("a", func(v string) {
			fmt.Println(v, "-", atomic.AddInt64(&x, 1))
		})
	}()

	time.Sleep(time.Second * 20000)
}

func TestDelay(t *testing.T) {
	//zhub.Delay("abx", "1", -1)

	//zhub.Publish("yk-topic", "hello yk.")

	for i := 0; i < 1000; i++ {
		zhub.Publish("a", "x-"+strconv.Itoa(i))

	}

	time.Sleep(time.Second * 5)
}

// 测试发送微信 模板消息
func TestWxSendMessage(t *testing.T) {
	tplData := map[string]interface{}{
		"appId": "wxa554ec3ab3bf1fc7",
		"templateData": toStr(map[string]interface{}{
			"touser":      "oIubq6svTJAoYTkos3KQHTPI1pYM",
			"template_id": "YKXh0EnmtE3y0RxZB8glRiKyYBlHv4DOf9IvTz27TGQ",
			"miniprogram": map[string]string{
				"appid": "wx8d2b5d48b91ca21d",
				"path":  "pages/main/mine/mine?type=2",
			},
			"data": map[string]interface{}{
				"first": map[string]string{
					"value": "您已成功报名比赛！",
					"color": "#173177",
				},
				"keyword1": map[string]string{
					"value": "高尔夫挑战赛",
					"color": "#173177",
				},
				"keyword2": map[string]string{
					"value": "2021.10.25 09:00 - 2021.10.25 18:00",
					"color": "#FF4777",
				},
				"keyword3": map[string]string{
					"value": "小东门体育馆",
					"color": "#173177",
				},
				"remark": map[string]string{
					"value": "感谢您的报名，请做好参赛准备。",
					"color": "#173177",
				},
			},
		}),
	}

	log.Println(toStr(tplData["templateData"]))

	zhub.Publish("wx:send-template-message", toStr(tplData))
}

// 监听各项目所有接口请求失败信息《接口请求失败，发送失败信息到 pro.app-error 主题》
func TestAppErrorConsole(t *testing.T) {
	zhub.Subscribe("app-error", func(v string) {
		strs := []string{
			"未登陆", "账号已在其他设备登录", "今日已签到", "您的登录状态已过期", "用户不存在", "暂无可预定场馆",
		}

		for _, str := range strs {
			if strings.Contains(v, str) {
				return
			}
		}
		log.Println(v)
	})

	time.Sleep(time.Hour * 8)
}

// -----------   rpc test   -----------

func TestRpcCall_S(t *testing.T) {
	zhub := newClient("zhub-cli-a", "x")
	zhub.RpcSubscribe("ai-test", func(r Rpc) RpcResult {
		upper := strings.ToUpper(r.Value)
		return RpcResult{Retcode: 0, Retinfo: "操作成功", Result: upper}
	})

	time.Sleep(time.Hour)
}

func TestRpcCall_C(t *testing.T) {
	/*zhub := newClient("dev_snow_x", "x")
	zhub.Rpc("a", "asdf", func(result RpcResult) {
		if result.Retcode != 0 {
			log.Println(result.Retinfo)
			return
		}

		log.Println(result.Result)
	})*/

	/*zhub.Rpc("h5publish:h5-dev", "m-sdk22-dev", func(res RpcResult) {
		print(res.Result)
	})*/

	zhub.Subscribe("zcore:monitor-error", func(v string) {
		fmt.Println(v)
	})

	time.Sleep(time.Hour)

	/*zhub.Rpc("zcore:file:up-token", "{'filetype':'file','limit':1}", func(res RpcResult) {
		fmt.Println(res)
	})*/
}

func TestBannedTalk(t *testing.T) {
	/*zhub.Rpc("im:banned-talk", "{'imtoken':'74074f9e599947ca940e71a9788e768f'}", func(res RpcResult) {
		fmt.Print(res)
	})*/

	encoding := base64.Encoding{}
	toString := encoding.EncodeToString([]byte("420101190001011234"))
	fmt.Println(toString)
}
