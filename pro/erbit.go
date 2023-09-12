// 浏览器url后面添加eggbox/saddle 会出现对应的图案,默认是雪堆状
package main

import (
	"bufio"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

const (
	width, height = 600, 320            //画布大小
	cells         = 100                 //单元格的个数
	xyrange       = 30.0                //坐标轴的范围(-xyrnage..+xyrange)
	xyscale       = width / 2 / xyrange //x或y轴上每个单位长度的像素
	zscale        = height * 0.5        //z轴上每个单位长度的像素
	angle         = math.Pi / 6         //x、y轴的角度(=30°)
)

var sin30, cos30 = math.Sin(angle), math.Cos(angle) // sin(30°),cos(30°)
type zFunc func(x, y float64) float64

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	surface(w, "f")
}
func eggboxs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	surface(w, "eggbox")
}
func saddles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	surface(w, "saddle")
}

func surface(w io.Writer, fName string) {
	var fn zFunc
	switch fName {
	case "saddle":
		fn = saddle
	case "eggbox":
		fn = eggbox
	default:
		fn = f
	}
	z_min, z_max := min_max(fn)
	fmt.Fprintf(w, "<svg xmlns='http://www.w3.org/2000/svg' "+
		"style='fill: white; stroke-width:0.7' "+
		"width='%d' height='%d'>", width, height)
	for i := 0; i < cells; i++ {
		for j := 0; j < cells; j++ {
			ax, ay := corner(i+1, j, fn)
			bx, by := corner(i, j, fn)
			cx, cy := corner(i, j+1, fn)
			dx, dy := corner(i+1, j+1, fn)
			if math.IsNaN(ax) || math.IsNaN(ay) || math.IsNaN(bx) || math.IsNaN(by) || math.IsNaN(cx) || math.IsNaN(cy) || math.IsNaN(dx) || math.IsNaN(dy) {
				continue
			} else {
				fmt.Fprintf(w, "<polygon style='stroke: %s;' points='%g,%g %g,%g %g,%g %g,%g'/>\n",
					color(i, j, z_min, z_max), ax, ay, bx, by, cx, cy, dx, dy)
			}
		}
	}
	fmt.Fprintln(w, "</svg>")
}

// minmax返回给定x和y的最小值/最大值并假设为方域的z的最小值和最大值。
func min_max(f zFunc) (min, max float64) {
	min = math.NaN()
	max = math.NaN()
	for i := 0; i < cells; i++ {
		for j := 0; j < cells; j++ {
			for xoff := 0; xoff <= 1; xoff++ {
				for yoff := 0; yoff <= 1; yoff++ {
					x := xyrange * (float64(i+xoff)/cells - 0.5)
					y := xyrange * (float64(j+yoff)/cells - 0.5)
					z := f(x, y)
					if math.IsNaN(min) || z < min {
						min = z
					}
					if math.IsNaN(max) || z > max {
						max = z
					}
				}
			}
		}
	}
	return min, max
}

func color(i, j int, zmin, zmax float64) string {
	min := math.NaN()
	max := math.NaN()
	for xoff := 0; xoff <= 1; xoff++ {
		for yoff := 0; yoff <= 1; yoff++ {
			x := xyrange * (float64(i+xoff)/cells - 0.5)
			y := xyrange * (float64(j+yoff)/cells - 0.5)
			z := f(x, y)
			if math.IsNaN(min) || z < min {
				min = z
			}
			if math.IsNaN(max) || z > max {
				max = z
			}
		}
	}

	color := ""
	if math.Abs(max) > math.Abs(min) {
		red := math.Exp(math.Abs(max)) / math.Exp(math.Abs(zmax)) * 255
		if red > 255 {
			red = 255
		}
		color = fmt.Sprintf("#%02x0000", int(red))
	} else {
		blue := math.Exp(math.Abs(min)) / math.Exp(math.Abs(zmin)) * 255
		if blue > 255 {
			blue = 255
		}
		color = fmt.Sprintf("#0000%02x", int(blue))
	}
	return color
}

func corner(i, j int, f zFunc) (float64, float64) {
	//求出网格单元(i,j)的顶点坐标(x,y)
	x := xyrange * (float64(i)/cells - 0.5)
	y := xyrange * (float64(j)/cells - 0.5)

	//计算曲面高度z
	z := f(x, y)
	//将(x, y, z)等角投射到二维SVG绘图平面上,坐标是(sx, sy)
	sx := width/2 + (x-y)*cos30*xyscale
	sy := height/2 + (x+y)*sin30*xyscale - z*zscale
	return sx, sy
}

func f(x, y float64) float64 {
	r := math.Hypot(x, y) //到(0,0)的距离
	return math.Sin(r) / r
}

func eggbox(x, y float64) float64 { //鸡蛋盒
	r := 0.2 * (math.Cos(x) + math.Cos(y))
	return r
}

func saddle(x, y float64) float64 { //马鞍
	a := 25.0
	b := 17.0
	a2 := a * a
	b2 := b * b
	r := y*y/a2 - x*x/b2
	return r
}

func sha256a(a string, b string) int {

	output1, err := strconv.ParseInt(a, 10, 64)
	output2, err := strconv.ParseInt(b, 10, 64)
	if err != nil {
		fmt.Print("数据错误")
	}
	c := []string{strconv.FormatInt(output1, 2)}
	d := []string{strconv.FormatInt(output2, 2)}
	n := 0
	for i := 0; i < len(c)-1; i++ {
		if c[i] == d[i] {
			continue
		}
		n++
	}
	return n
}

var hashmethod = flag.String("s", "", "查询")
var hashmethod1 = flag.String("a", "", "添加")

func pahash(flag string, str string) {
	if flag == "SHA256" {
		fmt.Print(sha256.Sum256([]byte(str)))
	}
	if flag == "SHA512" {
		fmt.Print(sha512.Sum512([]byte(str)))
	}
}

const lenth = 10

func rever(arr *[lenth]int) *[10]int {
	for i := 0; i < lenth/2; i++ {
		(*arr)[i], (*arr)[lenth-1-i] = (*arr)[lenth-1-i], (*arr)[i]
	}
	return arr
}
func rotate(arr [lenth]int, a int) [10]int {
	for i := 0; i < lenth; i++ {
		if i <= lenth-1-a {
			(arr)[i] = (arr)[i+a]
		} else if i > lenth-1-a {
			(arr)[i] = (arr)[i+a-lenth]
		}

	}
	return arr
}

func rangeRotate(slice []int, k int) []int {
	res := make([]int, len(slice))
	for i, val := range slice {
		res[(i+k)%len(slice)] = val
		fmt.Print(i)
	}
	return res

}

func delspace(strings []string) []string {
	for i := 1; i < len(strings); {
		if strings[i] == strings[i-1] {
			copy(strings[i:], strings[i+1:])
			strings = strings[:len(strings)-1]
		}
		i++
	}
	return strings
}

func equal(x, y map[string]int) bool {
	if len(x) != len(y) {
		return false
	}
	for k, xv := range x {
		if yv, ok := y[k]; !ok || yv != xv {
			return false
		}
	}
	return true
}

func wordfreq() {
	hash := make(map[string]int, 20)
	input := bufio.NewScanner(os.Stdin)
	input.Split(bufio.ScanWords)
	for input.Scan() {
		hash[input.Text()]++
	}
	fmt.Printf("%v", hash)
}

var prereqs = map[string][]string{
	"algorithms": {"data structures"},
	"calculus":   {"linear algebra"},
	"compilers": {
		"data structures",
		"formal languages",
		"computer organization",
	},
	"data structures":       {"discrete math"},
	"databases":             {"data structures"},
	"discrete math":         {"intro to programming"},
	"formal languages":      {"discrete math"},
	"networks":              {"operating systems"},
	"operating systems":     {"data structures", "computer organization"},
	"programming languages": {"data structures", "computer organization"},
}

func topoSort(m map[string][]string) []string {
	var order []string
	seen := make(map[string]bool)
	var visitAll func(items []string)
	visitAll = func(items []string) {
		for _, item := range items {
			fmt.Printf("元素为%s\t", item)
			if !seen[item] {
				seen[item] = true
				fmt.Printf("map元素为%s\t", m[item])
				visitAll(m[item])

				order = append(order, item)
				fmt.Printf("数组为：%s\t\n", order)
			}
		}
	}
	var keys []string
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	// fmt.Print(keys)
	visitAll(keys)
	return order
}

func topoSortMap(m map[string][]string) []string {
	var order []string
	seen := make(map[string]bool)
	var visitAll func(map[string][]string)
	visitAll = func(mp map[string][]string) {
		mp1 := make(map[string][]string, len(m))
		for k, v := range mp {
			fmt.Printf("k=%s\tv=%v\n", k, v)
			if !seen[k] {
				seen[k] = true
				for _, v1 := range v { // 遍历预修课程
					if !seen[v1] { // 判断预修课程是否被观测。由于图中某些节点存在多个边，因此，如果预修课程被提前观测，则不再观测该课程。
						value, ok := m[v1]
						fmt.Printf("va=%s\t%v\n", value, ok) // 判断预修课程是否存在学习序列
						if ok {
							mp1[v1] = value // 存在则继续观测
							fmt.Printf("mp1=%s\t\n", mp1)
						} else {
							mp1[v1] = make([]string, 0) // 不存在则结束观测
						}
					}
				}
				visitAll(mp1)
				order = append(order, k)
				fmt.Printf("order=%s\t\n", order)
			}
		}
	}
	visitAll(m)

	return order
}

func main() {

	atime := time.Now()
	var a = 1
	b := a + 1
	time.Sleep(10 * time.Second)
	fmt.Print(b, time.Since(atime))

}

func hasCycle(m map[string][]string) bool {
	// visited 记录已经访问过的节点
	visited := make(map[string]bool)
	// dfs 函数用来进行深度优先搜索
	var dfs func(string, map[string]bool) bool
	dfs = func(item string, path map[string]bool) bool {
		// 如果当前节点已经被访问过，则返回 false
		if visited[item] {
			return false
		}
		// 将当前节点标记为已访问，并加入访问路径
		visited[item] = true
		path[item] = true
		// 遍历当前节点的所有相邻节点
		for _, next := range m[item] {

			// 如果相邻节点已经在访问路径上，说明存在环，返回 true
			if path[next] || dfs(next, path) {
				return true
			}
		}
		// 将当前节点从访问路径中移除，并返回 false
		delete(path, item)
		return false
	}
	// 遍历 map 中的每个节点，并调用 dfs 函数检查是否存在环
	for k := range m {
		path := make(map[string]bool)
		if dfs(k, path) {
			return true
		}
	}
	// 如果所有节点都检查完，则说明图中不存在环，返回 false
	return false
}

type IntSet struct {
	words []uint64
}

// Has reports whether the set contains the non-negative value x.
func (s *IntSet) Has(x int) bool {
	word, bit := x/64, uint(x%64)
	return word < len(s.words) && s.words[word]&(1<<bit) != 0
}

// Add adds the non-negative value x to the set.
func (s *IntSet) Add(x int) {
	word, bit := x/64, uint(x%64)
	for word >= len(s.words) {
		s.words = append(s.words, 0)
	}
	s.words[word] |= 1 << bit
}

// UnionWith sets s to the union of s and t.
func (s *IntSet) UnionWith(t *IntSet) {
	for i, tword := range t.words {
		if i < len(s.words) {
			s.words[i] |= tword
		} else {
			s.words = append(s.words, tword)
		}
	}
}
