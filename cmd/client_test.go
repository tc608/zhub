package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var hub *ZHubClient
var once = sync.Once{}

func init() {
	once.Do(func() {
		hub := &ZHubClient{
			appname: "hub-cli",
			addr:    "127.0.0.1:1216",
			groupid: "C-0",
			auth:    "admin@123456",
		}
		err := hub.Init()

		if err != nil {
			log.Fatal(err)
		}
		hub = hub
	})
}

func TestLock(t *testing.T) {
	hub.Subscribe("lock", func(v string) {

	})

	var fun = func(x string) {
		log.Println("lock", time.Now().UnixNano()/1e6)
		lock := hub.Lock("a", 30)
		defer hub.Unlock(lock)
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

// 特殊符号测试
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
	return string(bs)
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

	hub.Publish("wx:send-template-message", toStr(tplData))
}

// 监听各项目所有接口请求失败信息《接口请求失败，发送失败信息到 pro.app-error 主题》
func TestAppErrorConsole(t *testing.T) {
	hub.Subscribe("app-error", func(v string) {
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
	hub.RpcSubscribe("ai-test", func(r Rpc) RpcResult {
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

	hub.Subscribe("zcore:monitor-error", func(v string) {
		fmt.Println(v)
	})

	time.Sleep(time.Hour)

	/*zhub.Rpc("zcore:file:up-token", "{'filetype':'file','limit':1}", func(res RpcResult) {
		fmt.Println(res)
	})*/
}
