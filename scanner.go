package main

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// 清除ARP表
func clearArpTable() error {
	cmd := exec.Command("netsh", "interface", "ip", "delete", "arpcache")
	return cmd.Run()
}

// 解析CIDR網段，返回所有IP地址
func parseCIDR(cidr string) ([]net.IP, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for ip := ipnet.IP.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		// 跳過網路位址和廣播位址
		if ip.Equal(ipnet.IP) || ip.Equal(broadcast(ipnet)) {
			continue
		}
		ips = append(ips, net.IP(make([]byte, len(ip))))
		copy(ips[len(ips)-1], ip)
	}
	return ips, nil
}

// IP地址遞增
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// 計算廣播地址
func broadcast(n *net.IPNet) net.IP {
	ip := make(net.IP, len(n.IP))
	copy(ip, n.IP)
	for i := 0; i < len(n.IP); i++ {
		ip[i] |= ^n.Mask[i]
	}
	return ip
}

// TCP連接掃描
func tcpScan(ip string, ports []string) {
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", ip+":"+port, 1*time.Second)
		if err == nil {
			conn.Close()
		}
	}
}

// Ping掃描
func pingScan(ip string) {
	cmd := exec.Command("ping", "-n", "1", "-w", "1000", ip)
	cmd.Run() // 不管成功失敗，都可能產生ARP記錄
}

// 讀取ARP表
func readArpTable() ([]ArpEntry, error) {
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseArpOutput(string(output), "", "0.0.0.0/0")
}

// 解析ARP表輸出，並過濾指定網段
func parseArpOutput(output string, scanBatch string, networkCIDR string) ([]ArpEntry, error) {
	var entries []ArpEntry

	// 解析網段以用於過濾
	_, ipnet, err := net.ParseCIDR(networkCIDR)
	if err != nil {
		return nil, fmt.Errorf("解析網段失敗: %v", err)
	}

	// ARP表格式: IP地址 MAC地址 類型
	// 例如: 192.168.1.1    00-11-22-33-44-55     dynamic
	re := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)\s+([0-9a-fA-F]{2}-[0-9a-fA-F]{2}-[0-9a-fA-F]{2}-[0-9a-fA-F]{2}-[0-9a-fA-F]{2}-[0-9a-fA-F]{2})`)

	matches := re.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			ip := match[1]
			mac := strings.ToUpper(strings.ReplaceAll(match[2], "-", ":"))

			// 檢查IP是否在指定網段內
			parsedIP := net.ParseIP(ip)
			if parsedIP == nil {
				continue
			}

			// 只保留指定網段內的IP
			if ipnet.Contains(parsedIP) {
				entry := ArpEntry{
					IP:        ip,
					MAC:       mac,
					Timestamp: time.Now(),
					ScanBatch: scanBatch,
				}
				entries = append(entries, entry)
			}
		}
	}

	return entries, nil
}

// ARP掃描 (Windows下使用ping實現)
func arpScan(cidr string, scanBatch string) ([]ArpEntry, error) {
	// 解析網段
	ips, err := parseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("解析網段失敗: %v", err)
	}

	// 清除ARP表
	if err := clearArpTable(); err != nil {
		return nil, fmt.Errorf("清除ARP表失敗: %v", err)
	}

	// 並發ping所有IP
	for _, ip := range ips {
		go pingScan(ip.String())
	}

	// 等待ping完成
	time.Sleep(3 * time.Second)

	// 讀取ARP表並添加批次ID
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("讀取ARP表失敗: %v", err)
	}

	return parseArpOutput(string(output), scanBatch, cidr)
}

// 執行網路掃描
func performNetworkScan(config NetworkConfig) ([]ArpEntry, error) {
	// 生成掃描批次ID (格式: YYYYMMDD-HHMMSS)
	scanBatch := time.Now().Format("20060102-150405")

	switch config.ScanMethod {
	case "arp-scan":
		return arpScan(config.NetworkCIDR, scanBatch)

	case "ping":
		// 1. 清除ARP表
		if err := clearArpTable(); err != nil {
			return nil, fmt.Errorf("清除ARP表失敗: %v", err)
		}

		// 2. 解析網段
		ips, err := parseCIDR(config.NetworkCIDR)
		if err != nil {
			return nil, fmt.Errorf("解析網段失敗: %v", err)
		}

		// 3. Ping方式掃描
		for _, ip := range ips {
			go pingScan(ip.String())
		}

		// 4. 等待掃描完成
		time.Sleep(5 * time.Second)

		// 5. 讀取ARP表
		cmd := exec.Command("arp", "-a")
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("讀取ARP表失敗: %v", err)
		}

		return parseArpOutput(string(output), scanBatch, config.NetworkCIDR)

	// case "tcp":
	// 	// 1. 清除ARP表
	// 	if err := clearArpTable(); err != nil {
	// 		return nil, fmt.Errorf("清除ARP表失敗: %v", err)
	// 	}
	//
	// 	// 2. 解析網段
	// 	ips, err := parseCIDR(config.NetworkCIDR)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("解析網段失敗: %v", err)
	// 	}
	//
	// 	// 3. TCP連接方式掃描
	// 	ports := strings.Split(config.TCPPorts, ",")
	// 	for i := range ports {
	// 		ports[i] = strings.TrimSpace(ports[i])
	// 	}
	//
	// 	for _, ip := range ips {
	// 		go tcpScan(ip.String(), ports)
	// 	}
	//
	// 	// 4. 等待掃描完成
	// 	time.Sleep(3 * time.Second)
	//
	// 	// 5. 讀取ARP表
	// 	cmd := exec.Command("arp", "-a")
	// 	output, err := cmd.Output()
	// 	if err != nil {
	// 		return nil, fmt.Errorf("讀取ARP表失敗: %v", err)
	// 	}
	//
	// 	return parseArpOutput(string(output), scanBatch, config.NetworkCIDR)

	default:
		return nil, fmt.Errorf("不支援的掃描方式: %s", config.ScanMethod)
	}
}
