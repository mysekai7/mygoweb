package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

var (
	AutoRender bool = true
)

type controllerInfo struct {
	regex          *regexp.Regexp
	params         map[int]string
	controllerType reflect.Type
}

type MyMux struct {
	routers []*controllerInfo
}

type Context struct {
	Params         map[string]string
	Request        *http.Request
	ResponseWriter http.ResponseWriter
}

type Controller struct {
	Ct        *Context
	Data      map[interface{}]interface{}
	ChildName string
	TplNames  string
}

func (c *Controller) Init(ct *Context, cn string) {
	c.Data = make(map[interface{}]interface{})
	c.TplNames = ""
	c.Ct = ct
	c.ChildName = cn
	//fmt.Println("struct v%", c)
}

func (c *Controller) Prepare() {

}

func (c *Controller) Get() {
	http.Error(c.Ct.ResponseWriter, "Method Not Allowed", 405)
}

func (c *Controller) Render() error {
	t, err := template.ParseFiles(c.TplNames)
	err = t.Execute(c.Ct.ResponseWriter, c.Data)
	if err != nil {
		//
	}
	return nil
}

type ControllerInterface interface {
	Init(ct *Context, cn string)
	Prepare()
	Get()
	Render() error
}

func (p *MyMux) AddRouter(pattern string, c ControllerInterface) {
	parts := strings.Split(pattern, "/")

	j := 0
	params := make(map[int]string)
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			expr := "([^/]+)"

			//a user may choose to override the defult expression
			// similar to expressjs: ‘/user/:id([0-9]+)’

			if index := strings.Index(part, "("); index != -1 {
				expr = part[index:]
				part = part[:index]
			}
			params[j] = part
			parts[i] = expr
			j++
		}
	}

	//recreate the url pattern, with parameters replaced
	//by regular expressions. then compile the regex

	pattern = strings.Join(parts, "/")
	regex, regexErr := regexp.Compile(pattern)
	if regexErr != nil {

		//TODO add error handling here to avoid panic
		panic(regexErr)
		return
	}

	//now create the Route
	t := reflect.Indirect(reflect.ValueOf(c)).Type()
	//fmt.Println(reflect.ValueOf(c))
	route := &controllerInfo{}
	route.regex = regex
	route.params = params
	route.controllerType = t
	//fmt.Println(t)
	p.routers = append(p.routers, route)

}

func (p *MyMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("------------------------------------------")

	var started bool = false
	requestPath := r.URL.Path
	for _, route := range p.routers {
		//check if Route pattern matches url
		if !route.regex.MatchString(requestPath) {
			//fmt.Println("Dont match url: " + requestPath)
			continue
		}

		//get submatches (params)
		matches := route.regex.FindStringSubmatch(requestPath)
		//fmt.Println("get submatches\t=> ", matches)

		//double check that the Route matches the URL pattern.
		if len(matches[0]) != len(requestPath) {
			continue
		}

		params := make(map[string]string)
		if len(route.params) > 0 {
			//add url parameters to the query param map
			values := r.URL.Query()

			for i, match := range matches[1:] {
				values.Add(route.params[i], match)
				params[route.params[i]] = match
				//fmt.Println("match =>", match)
				//fmt.Println("route.params[i]\t=> ", route.params[i])
			}
			//fmt.Println("values\t=> ", values)
			//reassemble query params and add to RawQuery
			r.URL.RawQuery = url.Values(values).Encode() + "&" + r.URL.RawQuery
			//r.URL.RawQuery = url.Values(values).Encode()
			//fmt.Println("r.URL.RawQuery\t=> ", r.URL.RawQuery)
		}
		//fmt.Println("params\t=> ", params)

		//Invoke the request handler
		vc := reflect.New(route.controllerType)
		fmt.Println("vc\t=> ", vc)
		init := vc.MethodByName("Init")
		fmt.Println("init\t=> ", init)
		ct := &Context{ResponseWriter: w, Request: r, Params: params}
		fmt.Println("ct\t=> ", ct)
		in := make([]reflect.Value, 2)
		fmt.Println("in\t=> ", in)
		in[0] = reflect.ValueOf(ct)
		in[1] = reflect.ValueOf(route.controllerType.Name())
		fmt.Println("in\t=> ", in)
		init.Call(in)
		in = make([]reflect.Value, 0)
		method := vc.MethodByName("Prepare")
		method.Call(in)
		if r.Method == "GET" {
			method = vc.MethodByName("Get")
			method.Call(in)
		}
		if AutoRender {
			method = vc.MethodByName("Render")
			method.Call(in)
		}
		started = true
		break
	}

	if started == false {
		http.NotFound(w, r)
	}
}

func sayhelloName(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello myroute!")
}

func NewMyMux() *MyMux {
	return &MyMux{routers: make([]*controllerInfo, 0)}
}

type MainController struct {
	Controller
}

func (this *MainController) Get() {
	this.Data["Username"] = "chao.wang"
	this.Data["Email"] = "mysekai7@gmail.com"
	this.TplNames = "index.tpl"
	fmt.Println(this.Ct.Params)
}

type ProfileController struct {
	Controller
}

func (this *ProfileController) Get() {
	this.Ct.Request.ParseForm()
	fmt.Println(this.Ct.Request.Form.Get("page"))
	this.Data["Username"] = this.Ct.Request.Form.Get("username")
	this.Data["Email"] = "907813456@qq.com"
	this.TplNames = "index.tpl"
	fmt.Println(this.Ct.Params)
}

func main() {
	mux := NewMyMux()
	mux.AddRouter("/", &MainController{})
	mux.AddRouter("/:param", &MainController{})
	mux.AddRouter("/profile/:uid([0-9]+)", &ProfileController{})

	http.ListenAndServe(":9090", mux)
}
