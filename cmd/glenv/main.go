package main

import (
 "flag"
 "fmt"
 "os"
 "os/exec"
 "strings"

 "github.com/angyedz/veyra-glenv/internal/elfinfo"
 "github.com/angyedz/veyra-glenv/internal/launcher"
 glruntime "github.com/angyedz/veyra-glenv/internal/runtime"
)

const version="0.2.0-dev"
func main(){os.Exit(run(os.Args[1:]))}
func rootFlag(fs *flag.FlagSet)*string{
 initial:=os.Getenv("GLENV_RUNTIME")
 if initial==""{initial="/run/glenv/runtime"}
 return fs.String("runtime",initial,"glibc runtime prefix or sysroot")
}
func run(args []string)int{
 if len(args)==0{usage();return 2}
 switch args[0]{
 case "version":fmt.Println("veyra-glenv",version);return 0
 case "inspect":
  if len(args)!=2{fmt.Fprintln(os.Stderr,"usage: glenv inspect <ELF>");return 2}
  b,err:=elfinfo.Inspect(args[1]);if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  fmt.Printf("file: %s\narchitecture: %s\ninterpreter: %s\nglibc: %t\n",b.Path,b.Machine,b.Interpreter,b.Glibc())
  return 0
 case "doctor":
  fs:=flag.NewFlagSet("doctor",flag.ContinueOnError);root:=rootFlag(fs)
  if err:=fs.Parse(args[1:]);err!=nil{return 2}
  r,err:=glruntime.Detect(*root);if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  fmt.Println("runtime:",r.Root);fmt.Println("sysroot:",r.Sysroot);fmt.Println("loader:",r.Loader)
  return 0
 case "run":
  fs:=flag.NewFlagSet("run",flag.ContinueOnError);root:=rootFlag(fs)
  dry:=fs.Bool("dry-run",false,"print loader invocation without executing")
  if err:=fs.Parse(args[1:]);err!=nil{return 2}
  rest:=fs.Args()
  if len(rest)==0{fmt.Fprintln(os.Stderr,"usage: glenv run [--runtime PATH] [--dry-run] <program> [args...]");return 2}
  plan,err:=launcher.PlanFor(*root,rest[0],rest[1:]);if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  if *dry{fmt.Println(strings.Join(plan.Args," "));return 0}
  command:=plan.Command()
  if err:=command.Run();err!=nil{
   var exitErr *exec.ExitError
   if ok:=asExit(err,&exitErr);ok{return exitErr.ExitCode()}
   fmt.Fprintln(os.Stderr,err);return 1
  }
  return 0
 default:fmt.Fprintln(os.Stderr,"unknown command:",args[0]);usage();return 2
 }
}
func asExit(err error,target **exec.ExitError)bool{
 e,ok:=err.(*exec.ExitError);if ok{*target=e};return ok
}
func usage(){fmt.Fprintln(os.Stderr,"usage: glenv <version|inspect|doctor|run> [options]")}
