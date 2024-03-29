package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func callFn(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fn := vars["fn"]
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "fn/" + fn,
	}, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	normalOut, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(normalOut)
	w.Write(body[7:])
}

type codeFn struct {
	filename   string
	dockerfile string
}

func createImage(r *http.Request, codeFn codeFn) {
	vars := mux.Vars(r)
	fn := vars["fn"]
	body, err := ioutil.ReadAll(r.Body)
	check(err)

	os.MkdirAll("/tmp/"+fn, os.ModePerm)

	err = ioutil.WriteFile("/tmp/"+fn+"/"+codeFn.filename, body, 0644)
	check(err)

	d1 := []byte(codeFn.dockerfile)
	err = ioutil.WriteFile("/tmp/"+fn+"/Dockerfile", d1, 0644)
	check(err)

	Tar("/tmp/"+fn, "/tmp/"+fn+".tar")

	buildOptions := types.ImageBuildOptions{
		Tags:           []string{"fn/" + fn},
		Dockerfile:     "Dockerfile",
		SuppressOutput: true,
		Remove:         true,
		ForceRemove:    true,
		PullParent:     true,
	}

	dockerBuildContext, err := os.Open("/tmp/" + fn + ".tar")
	defer dockerBuildContext.Close()

	if err != nil {
		log.Fatalf("build context failed:%v", err)
	}

	cli, err := client.NewEnvClient()
	buildResponse, err := cli.ImageBuild(context.Background(), dockerBuildContext, buildOptions)
	if err != nil {
		log.Fatalf("buildImage=%s failed:%v", "fn/"+fn, err)
	}

	defer buildResponse.Body.Close()
	io.Copy(os.Stdout, buildResponse.Body)

	err = os.Remove("/tmp/" + fn + ".tar")
	check(err)
	err = os.RemoveAll("/tmp/" + fn)
	check(err)
}

func nodeFn(w http.ResponseWriter, r *http.Request) {
	createImage(r, codeFn{
		filename:   "index.js",
		dockerfile: "FROM node:alpine\nWORKDIR /app\nCOPY index.js /app/\nENTRYPOINT [\"node\", \"/app/index.js\"]",
	})

	fmt.Fprintf(w, "FN Ready")
}

func rubyFn(w http.ResponseWriter, r *http.Request) {
	createImage(r, codeFn{
		filename:   "script.rb",
		dockerfile: "FROM ruby:alpine\nWORKDIR /app\nCOPY script.rb /app/\nENTRYPOINT [\"ruby\", \"/app/script.rb\"]",
	})

	fmt.Fprintf(w, "FN Ready")
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/node/{fn}", nodeFn)
	r.HandleFunc("/ruby/{fn}", rubyFn)
	r.HandleFunc("/call/{fn}", callFn)
	log.Println("Listen on port 8080...")
	http.ListenAndServe(":8080", r)
}
