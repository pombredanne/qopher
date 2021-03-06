// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qopher

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"

	"qopher/task"
)

func init() {
	http.HandleFunc("/admin/add", adminAdd)
	http.HandleFunc("/admin/polldebug", adminPollDebug)
	http.HandleFunc("/admin/wipe", adminWipe)
}

func checkAdmin(rw http.ResponseWriter, r *http.Request) bool {
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil || !u.Admin {
		http.Error(rw, "requires admin", 401)
		return false
	}
	return true
}

func adminAdd(rw http.ResponseWriter, r *http.Request) {
	if !checkAdmin(rw, r) {
		return
	}
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	id := r.FormValue("id")
	if id == "" {
		http.Error(rw, "no id parameter", 400)
		return
	}
	typeId := "manual." + id
	k := datastore.NewKey(ctx, "Task", typeId, 0, nil)
	now := time.Now()
	task := &Task{
		Owner:    u.Email,
		Type:     "manual",
		ID:       id,
		Created:  now,
		Modified: now,
		Assigned: now,
	}
	_, err := datastore.Put(ctx, k, task)
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(rw, "put of %q = %v", typeId, err)
}

func adminWipe(rw http.ResponseWriter, r *http.Request) {
	if !checkAdmin(rw, r) {
		return
	}
	ctx := appengine.NewContext(r)
	q := datastore.NewQuery("Task").KeysOnly()
	keys, err := q.GetAll(ctx, nil)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	err = datastore.DeleteMulti(ctx, keys)
	if err != nil {
		http.Error(rw, err.Error(), 500)
	} else {
		fmt.Fprintf(rw, "success. wiped all %d tasks.", len(keys))
	}
}

func adminPollDebug(rw http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	m := Poll(&appengineEnv{ctx}, 25*time.Second, task.Types)
	var page struct {
		Types []PollResult
	}
	for _, tt := range task.Types {
		page.Types = append(page.Types, m[tt.Type()])
	}
	pollDebugPage.ExecuteTemplate(rw, "polldebug", &page)
}

var pollDebugPage = template.Must(template.New("polldebug").Funcs(template.FuncMap{
	"taskURL": func(taskType, id string) string {
		return task.TypeMap[taskType].TaskURL(id)
	},
}).Parse(`
<!doctype html>
<html>
  <head>
    <title>qopher poll debug</title>
  </head>
  <body>

<h1>qopher task poll debug</h1>

{{range $i, $tt := .Types}}
<h2>{{$tt.Type}}</h2>
  {{if $tt.Err}}
     <b>Error:</b> {{$tt.Err}}
  {{else}}
    {{range $i, $task := $tt.Tasks}}
       <p><b><a href="{{taskURL $tt.Type $task.ID}}">[ID {{$task.ID}}]</a></b> - {{$task.Title}}</p>
    {{end}}
  {{end}}

{{end}}

  </body>
</html>
`))
