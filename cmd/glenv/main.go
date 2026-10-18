package main

import (
 "errors"
 "flag"
 "fmt"
 "os"
 "path/filepath"

 glruntime "github.com/angyedz/veyra-glenv/internal/runtime"
)

const version = "0.1.0-dev"

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
 if len(args)==0 { usage(); return 2 }
 switch args[0] {
 case "version":
  fmt.Println("veyra-glenv",version)
  return 0
 case "doctor":
  fs:=flag.NewFlagSet("doctor",flag.ContinueOnError)
  root:=fs.String("runtime",os.Getenv("GLENV_RUNTIME"),"glibc runtime root")
  if err:=fs.Parse(args[1:]);err!=nil{return 2}
  if *root=="" { *root="/opt/glibc-env" }
  r,err:=glruntime.Detect(*root)
  if err!=nil {
   if errors.Is(err,glruntime.ErrNotFound) { fmt.Fprintln(os.Stderr,"runtime unavailable:",err) } else { fmt.Fprintln(os.Stderr,"runtime error:",err) }
   return 1
  }
  fmt.Println("runtime:",r.Root)
  fmt.Println("loader:",filepath.Clean(r.Loader))
  return 0
 default:
  fmt.Fprintln(os.Stderr,"unknown command:",args[0])
  usage()
  return 2
 }
}

func usage() {
 fmt.Fprintln(os.Stderr,"usage: glenv <version|doctor> [options]")
}
