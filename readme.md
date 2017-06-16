## 目的

并发向指定的递归服务器查询域名的CNAME记录,确认站点域名是否在使用指定的CDN。

## 说明

可以从标准输入读取，也可以从指定的file文件读取。如果file没有指定，就是从标准输入读取。
如果查询出现异常，比如超时等、会重试直到所有域名都返回DNS应答。

1. batch 并发数
2. dns 递归服务器ip
3. file 输入域名文件、一行一个域名
4. port dns服务器端口
5. timeout 查询超时时间
6. verbose 输入信息的详细度

## 用法说明

```
Usage of ./dnsdetector:
  -batch int
    	concurrent number (default 100)
  -dns string
    	recursive dns server (default "114.114.114.114")
  -file string
    	domains file, read from stdin if not specify
  -port string
    	recursive dns server (default "53")
  -timeout int
    	timeout  (default 10)
  -verbose int
    	print verbose dns query info (default 1)

```

## 举例

命令
```
 ./cdn-domain-detector  -suffix=cdn.net. -file tempdomains
 or
 cat tempdomains | ./cdn-domain-detector  -suffix=cdn.net.

```

结果

```
qname count 8
start query  www.aiduanzi.cn
start query  img.jfzjt.com
start query  p.aiduanzi.cn
start query  baidu.com
start query  hao123.com
start query  www.432520.com.
www.432520.com.	410	IN	CNAME	2872b9a755fcb8b4.cdn.fhldns.com.
start query  case.bixenon.cn
case.bixenon.cn.	486	IN	CNAME	case.bixenon.cn.cname.yunjiasu-cdn.net.
case.bixenon.cn cname domain suffix is cdn.net.
start query  static.bixenon.cn
static.bixenon.cn.	410	IN	CNAME	static.bixenon.cn.cname.yunjiasu-cdn.net.
static.bixenon.cn cname domain suffix is cdn.net.
*****domains in cdn cdn.net. as follows******
case.bixenon.cn
static.bixenon.cn
******total 2 domains in cdn******
```
