package main

import (
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func runNodeFn(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fn := vars["fn"]
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "serverless/" + fn,
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

func nodeFn(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fn := vars["fn"]
	body, err := ioutil.ReadAll(r.Body)
	check(err)

	os.MkdirAll("/tmp/"+fn, os.ModePerm)

	err = ioutil.WriteFile("/tmp/"+fn+"/index.js", body, 0644)
	check(err)

	d1 := []byte("FROM node:alpine\nWORKDIR /app\nCOPY index.js /app/\nCMD node /app/index.js")
	err = ioutil.WriteFile("/tmp/"+fn+"/Dockerfile", d1, 0644)
	check(err)

	Tar("/tmp/"+fn, "/tmp/docker.tar")

	buildOptions := types.ImageBuildOptions{
		Tags:           []string{"serverless/" + fn},
		Dockerfile:     "Dockerfile",
		SuppressOutput: true,
		Remove:         true,
		ForceRemove:    true,
		PullParent:     true,
	}

	dockerBuildContext, err := os.Open("/tmp/docker.tar")
	defer dockerBuildContext.Close()

	if err != nil {
		log.Fatalf("build context failed:%v", err)
	}

	cli, err := client.NewEnvClient()
	buildResponse, err := cli.ImageBuild(context.Background(), dockerBuildContext, buildOptions)
	if err != nil {
		log.Fatalf("buildImage=%s failed:%v", "serverless/"+fn, err)
	}

	defer buildResponse.Body.Close()
	io.Copy(os.Stdout, buildResponse.Body)

	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/node/{fn}", nodeFn)
	r.HandleFunc("/call/{fn}", runNodeFn)
	log.Println("Listen on port 8080...")
	http.ListenAndServe(":8080", r)
}
