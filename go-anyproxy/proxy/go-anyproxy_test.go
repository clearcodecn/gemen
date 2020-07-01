package proxy

import (
	"fmt"
	"github.com/andybalholm/brotli"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
)

func TestDecodeResponse(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://www.zhihu.com", nil)
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Cookie", "_zap=2101b3b5-3031-4521-a2f8-378fc9f62979; d_c0=\"AIAk84AFCQ-PTvRNTaDBJfUgGtmTBoSK_eg=|1551083460\"; __gads=ID=28761033f7820554:T=1554195435:S=ALNI_MZiIiRWQQ-nYjAbzdC0VbyBT4YNAg; _ga=GA1.2.592822435.1583743649; ISSW=1; __utma=51854390.592822435.1583743649.1583919606.1583919606.1; __utmz=51854390.1583919606.1.1.utmcsr=baidu|utmccn=(organic)|utmcmd=organic|utmctr=ios%20%E7%BC%96%E7%A8%8B%E4%B9%A6%E7%B1%8D; __utmv=51854390.000--|3=entry_date=20190225=1; z_c0=\"2|1:0|10:1584340703|4:z_c0|92:Mi4xX1BsVUFnQUFBQUFBZ0NUemdBVUpEeVlBQUFCZ0FsVk4zMnhjWHdCVk9VQW90VXJhdFBQRmN6ak1rbjRnOURERktn|f165934a0b9322bbcd4990f720f5cd5454ee307ba6aab5c0a453d5afea01a7ed\"; tst=r; q_c1=c4d2f8619b9b4935ad00338d189b3989|1591166665000|1551083461000; Hm_lvt_98beee57fd2ef70ccdd5ca52b9740c49=1592286754,1592810615,1592810625,1592902101; _xsrf=hPUwaqo9wpLw4vfrp0V2fPCqiormsPRb; KLBRSID=37f2e85292ebb2c2ef70f1d8e39c2b34|1593509358|1593509358")
	req.Header.Add("Cache-Control", "max-age=0")
	req.Header.Add("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Add("Sec-Fetch-Dest", "document")
	req.Header.Add("Sec-Fetch-Mode", "navigate")
	req.Header.Add("Sec-Fetch-Site", "none")
	req.Header.Add("Sec-Fetch-User", "?1")
	req.Header.Add("Upgrade-Insecure-Requests", "1")
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Fatalln(err)
	}
	for k := range resp.Header {
		fmt.Println(k, resp.Header.Get(k))
	}
	//io.Copy(os.Stdout,resp.Body)
	//var buf = bytes.NewBuffer(nil)
	//for {
	//	b := make([]byte, 20480)
	//	n, err := resp.Body.Read(b)
	//	if err != nil {
	//		if err == io.EOF {
	//			buf.Write(b[:n])
	//			break
	//		}
	//		log.Fatal(err)
	//	}
	//	buf.Write(b[:n])
	//}
	//fmt.Println(resp.Header,buf.String())
	//
	//fmt.Println(buf.Len())
	r := brotli.NewReader(resp.Body)
	//r, err := gzip.NewReader(resp.Body)
	d, _ := ioutil.ReadAll(r)
	fmt.Println(string(d))

	//_,err = decodeResponse(resp)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//data,err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	log.Fatal(err)
	//}
}
