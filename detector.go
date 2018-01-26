package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

var (
	domainfile = flag.String("file", "", "domains file, read from stdin if not specify")
	dnserver   = flag.String("dns", "114.114.114.114", "recursive dns server")
	port       = flag.String("port", "53", "recursive dns server")
	verbose    = flag.Int("verbose", 1, "print verbose dns query info")
	batchNum   = flag.Int("batch", 100, "concurrent number")
	timeout    = flag.Int("timeout", 10, "timeout ")
	suffix     = flag.String("suffix", "cn.", "which cdn does this domain used")
	retry      = flag.Int("retry", 3, "retry times after query failed")
)

func init() {
	flag.Parse()
}
func main() {
	qname := make([]string, 0, 100) //{"www.baidu.com", "www.chaoshanw.cn"}
	if *domainfile == "" {
		inputReader := bufio.NewReader(os.Stdin)
		for {
			input, err := inputReader.ReadString('\n')
			if err != nil {
				break
			}
			qname = append(qname, strings.Trim(input, "\n"))
		}
	} else {
		fp, err := os.Open(*domainfile)
		if err != nil {
			fmt.Printf("open %s failed\n", *domainfile)
			return
		}
		br := bufio.NewReader(fp)
		for {
			line, _, err := br.ReadLine()
			if err != nil {
				break
			}
			qname = append(qname, string(line))
		}
		fp.Close()
	}
	fmt.Printf("qname count %d\n", len(qname))

	nameserver := *dnserver + ":" + *port
	batch_query(qname, nameserver)
}

func query_one(nameserver, v string, control chan bool, cdnD, noanswerD, otherD, retryD *[]string) {
	c := new(dns.Client)
	c.Net = "udp"
	c.Timeout = time.Duration(*timeout) * time.Second
	c.DialTimeout = 10 * time.Second
	c.ReadTimeout = 5 * time.Second
	c.WriteTimeout = 5 * time.Second
	m := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative:     true,
			AuthenticatedData: true,
			CheckingDisabled:  false,
			RecursionDesired:  true,
			Opcode:            dns.OpcodeQuery,
		},
		Question: make([]dns.Question, 1),
	}
	qt := dns.TypeA
	qc := uint16(dns.ClassINET)
	m.Question[0] = dns.Question{Name: dns.Fqdn(v), Qtype: qt, Qclass: qc}
	m.Id = dns.Id()
	r, _, err := c.Exchange(m, nameserver)
	if *verbose >= 1 {
		fmt.Println("start query ", v)
	}
	if err != nil {
		fmt.Printf("%s error reason: %s\n", v, err)
		*retryD = append(*retryD, v)
	} else {
		if *verbose >= 3 {
			if r.Ns != nil && len(r.Ns) > 0 {
				fmt.Println(r.Ns[0].String())
			} else {
				fmt.Printf("domain %s has not ns record\n", v)
			}
		}
		r.Compress = true
		cnameCount := 0
		aCount := 0
		for _, rr := range r.Answer {
			//fmt.Printf("rr type %s\n", dns.Type(rr.Header().Rrtype).String())
			if dns.Type(rr.Header().Rrtype).String() == "A" {
				aCount++
			} else if dns.Type(rr.Header().Rrtype).String() == "CNAME" {
				cnameCount++
			}
		}
		fmt.Printf("@@@ %s msg size %d, cname %d, ipv4 %d\n------\n", dns.Fqdn(v), r.Len()+11, cnameCount, aCount)
		if len(r.Answer) >= 1 {
			dnsrr := r.Answer[0].String()
			if *verbose >= 1 {
				fmt.Println(dnsrr)
			}
			vv := strings.Split(dnsrr, "\t")
			result := vv[4]
			if strings.HasSuffix(result, dns.Fqdn(*suffix)) {
				if *verbose >= 1 {
					fmt.Printf("%s cname domain suffix is %s\n", v, *suffix)
				}
				*cdnD = append(*cdnD, fmt.Sprintf("%-40s %s", v, result))
			} else {
				*otherD = append(*otherD, fmt.Sprintf("%-40s %s", v, result))
			}
		} else {
			if len(r.Answer) == 0 {
				*noanswerD = append(*noanswerD, fmt.Sprintf("%-40s [no answer]", v))
			} else {
				*otherD = append(*otherD, fmt.Sprintf("%-40s %s", v, r.Answer[0].String()))
			}
		}
	}
	<-control
}
func batch_query(qname []string, nameserver string) {
	cdnDomains := make([]string, 0, 1000)
	noanswerDomains := make([]string, 0, 1000)
	otherDomains := make([]string, 0, 1000)
	retryDomains := make([]string, 0, 1000)
	control := make(chan bool, *batchNum)
	qnamelist := qname
	retryCount := 0

	for {
		for _, v := range qnamelist {
			go query_one(nameserver, v, control, &cdnDomains, &noanswerDomains, &otherDomains, &retryDomains)
			control <- true
		}
		if *verbose >= 2 {
			fmt.Printf("!!!!!!%d domains need retry!!!!!!\n", len(retryDomains))
		}
		if *verbose >= 3 {
			for _, d := range retryDomains {
				fmt.Printf("%s\n", d)
			}
		}
		if len(cdnDomains)+len(otherDomains)+len(noanswerDomains) < len(qname) {
			fmt.Printf("still has domain %d ...\n", len(retryDomains))
			fmt.Printf("no answer domains %d ...\n", len(noanswerDomains))
			fmt.Printf("other cdn domains %d ...\n", len(otherDomains))
			fmt.Printf("%s cdn domains %d ...\n", *suffix, len(cdnDomains))
			time.Sleep(time.Second)
			for {
				if len(cdnDomains)+len(otherDomains)+len(noanswerDomains)+len(retryDomains) != len(qname) {
					fmt.Printf("still has goroutine runing ...\n")
					time.Sleep(time.Second)
				} else {
					break
				}
			}
			qnamelist = retryDomains
			retryDomains = retryDomains[0:0]
			retryCount++
			if retryCount > *retry {
				fmt.Printf("read max retry times, break\n")
				break
			}
		} else {
			break
		}

	}
	if *verbose >= 1 {
		fmt.Printf("*****domains no answers as follows******\n")
		for _, d := range noanswerDomains {
			fmt.Println(d)
		}
		fmt.Printf("*****domains not in cdn %s as follows******\n", *suffix)
		for _, d := range otherDomains {
			fmt.Println(d)
		}
		fmt.Printf("*****domains in cdn %s as follows******\n", *suffix)
		for _, d := range cdnDomains {
			fmt.Println(d)
		}
	}
	fmt.Printf("******total %d domains no answer******\n", len(noanswerDomains))
	fmt.Printf("******total %d domains not in cdn******\n", len(otherDomains))
	fmt.Printf("******total %d domains in cdn******\n", len(cdnDomains))
}
