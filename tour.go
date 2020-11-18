package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"golang.org/x/tools/godoc/static"
	"golang.org/x/tools/present"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	uiContent      []byte
	lessonNotFound = fmt.Errorf("lesson not found")
	lessons        = make(map[string][]byte)
)

func initLessons(tmpl *template.Template, content string) error {
	dir, err := os.Open(content)
	if err != nil {
		return err
	}
	files, err := dir.Readdirnames(0)
	if err != nil {
		return err
	}
	for _, f := range files {
		if filepath.Ext(f) != ".article" {
			continue
		}
		content, err := parseLesson(tmpl, filepath.Join(content, f))
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(f, ".article")
		lessons[name] = content
	}
	return nil
}

type File struct {
	Name    string
	Content string
	Hash    string
}
type Page struct {
	Title   string
	Content string
	Files   []File
}
type Lesson struct {
	Title       string
	Description string
	Pages       []Page
}

func parseLesson(tmpl *template.Template, path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	doc, err := present.Parse(prepContent(f), path, 0)
	if err != nil {
		return nil, err
	}
	lesson := Lesson{
		doc.Title,
		doc.Subtitle,
		make([]Page, len(doc.Sections)),
	}
	for i, sec := range doc.Sections {
		p := &lesson.Pages[i]
		w := new(bytes.Buffer)
		if err := sec.Render(w, tmpl); err != nil {
			return nil, err
		}
		p.Title = sec.Title
		p.Content = w.String()
		codes := findPlayCode(sec)
		p.Files = make([]File, len(codes))
		for i, c := range codes {
			f := &p.Files[i]
			f.Name = c.FileName
			f.Content = string(c.Raw)
			hash := sha1.Sum(c.Raw)
			f.Hash = base64.StdEncoding.EncodeToString(hash[:])
		}
	}
	w := new(bytes.Buffer)
	if err := json.NewEncoder(w).Encode(lesson); err != nil {
		return nil, fmt.Errorf("encode lesson:%v", err)
	}
	return w.Bytes(), nil
}

func findPlayCode(e present.Elem) []*present.Code {
	var r []*present.Code
	switch v := e.(type) {
	case present.Code:
		if v.Play {
			r = append(r, &v)
		}
	case present.Section:
		for _, s := range v.Elem {
			r = append(r, findPlayCode(s)...)
		}
	}
	return r
}

func renderUI(w io.Writer) error {
	if uiContent != nil {
		panic("renderUI called before successful initTour")
	}
	_, err := w.Write(uiContent)
	return err
}

func initTour(root, transport string) error {
	present.PlayEnabled = true

	action := filepath.Join(root, "template", "action.tmpl")
	tmpl, err := present.Template().ParseFiles(action)
	if err != nil {
		return fmt.Errorf("parse template:%v", err)
	}
	contentPath := filepath.Join(root, "content")
	if err := initLessons(tmpl, contentPath); err != nil {
		return fmt.Errorf("init lessons:%v", err)
	}
	index := filepath.Join(root, "template", "index.html")
	ui, err := template.ParseFiles(index)
	if err != nil {
		return fmt.Errorf("template parseFiles:%v", err)
	}
	buf := new(bytes.Buffer)
	data := struct {
		AnalyticsHTML template.HTML
		SocketAddr    string
		Transport     template.JS
	}{analyticsHTML, socketAddr(), template.JS(transport)}
	if err := ui.Execute(buf, data); err != nil {
		return fmt.Errorf("render UI:%v", err)
	}
	uiContent = buf.Bytes()

	return initScript(root)
}
func initScript(root string) error {
	modTime := time.Now()
	b := new(bytes.Buffer)
	content, ok := static.Files["playground.js"]
	if !ok {
		return fmt.Errorf("playground js not found in static files")
	}
	b.WriteString(content)
	files := []string{
		"static/lib/jquery.min.js",
		"static/lib/jquery-ui.min.js",
		"static/lib/angular.min.js",
		"static/lib/codemirror/lib/codemirror.js",
		"static/lib/codemirror/mode/go/go.js",
		"static/lib/angular-ui.min.js",
		"static/js/app.js",
		"static/js/controllers.js",
		"static/js/directives.js",
		"static/js/services.js",
		"static/js/values.js",
	}
	for _, file := range files {
		f, err := ioutil.ReadFile(filepath.Join(root, file))
		if err != nil {
			return fmt.Errorf("could open file %v:%v", file, err)
		}
		_, err = b.Write(f)
		if err != nil {
			return fmt.Errorf("error concatenating %v:%v", file, err)
		}
	}
	http.HandleFunc("script.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-type", "application/javascript")
		w.Header().Set("Cache-control", "max-age=604800")
		http.ServeContent(w, r, "", modTime, bytes.NewReader(b.Bytes()))
	})
	return nil
}
func writeLesson(name string, w io.Writer) error {
	if uiContent == nil {
		panic("writeLesson called before successful initTour")
	}
	if len(name) == 0 {
		return writeAllLessons(w)
	}
	l, ok := lessons[name]
	if !ok {
		return lessonNotFound
	}
	_, err := w.Write(l)
	return err
}

func writeAllLessons(w io.Writer) error {
	if _, err := fmt.Fprint(w, "{"); err != nil {
		return err
	}
	nLessons := len(lessons)
	for k, v := range lessons {
		if _, err := fmt.Fprintf(w, "%q:%s", k, v); err != nil {
			return err
		}
		nLessons--
		if nLessons != 0 {
			if _, err := fmt.Fprint(w, ","); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprint(w, "}")
	return err
}
