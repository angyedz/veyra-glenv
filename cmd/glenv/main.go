package main

import (
 "flag"
 "fmt"
 "os"
 "os/exec"
 "strings"
 "encoding/json"
 "path/filepath"
 "github.com/angyedz/veyra-glenv/internal/deployment"

 "github.com/angyedz/veyra-glenv/internal/elfinfo"
 "github.com/angyedz/veyra-glenv/internal/launcher"
 glruntime "github.com/angyedz/veyra-glenv/internal/runtime"
)

const version="0.3.0-dev"
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
 case "release":
  fs:=flag.NewFlagSet("release",flag.ContinueOnError)
  storeRoot:=fs.String("store","/var/lib/veyra/glenv","version store (requires administrator ownership)")
  if err:=fs.Parse(args[1:]);err!=nil{return 2}
  rest:=fs.Args()
  if len(rest)==0{fmt.Fprintln(os.Stderr,"usage: glenv release [--store PATH] <prepare-copy|inventory|verify-signed|verify|stage|activate|pin|pair-host|pair-status|boot-pair-pin|bind-host|boot-pin|boot-status|rollback|status> ...");return 2}
  store:=deployment.Store{Root:*storeRoot}
  switch rest[0]{
  case "prepare-copy":
   if len(rest)!=4{return 2}
   if err:=deployment.PrepareCopy(rest[1],rest[2],rest[3]);err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "inventory":
   if len(rest)!=3{return 2}
   m,err:=deployment.Inventory(rest[1],rest[2]);if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
   encoded,err:=json.MarshalIndent(m,"","  ");if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
   fmt.Println(string(encoded));return 0
  case "verify-signed":
   if len(rest)!=3{return 2}
   key,err:=deployment.ReadPublicKey(rest[2]);if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
   manifest,err:=deployment.VerifySigned(rest[1],key);if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
   fmt.Printf("verified signed glenv release: %s\n",manifest.Version)
  case "verify":
   if len(rest)!=2{return 2}
   if err:=store.Verify(rest[1]);err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "stage":
   if len(rest)!=3{return 2}
   if err:=store.Stage(rest[1],rest[2]);err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "activate":
   if len(rest)!=2{return 2}
   if err:=store.Activate(rest[1]);err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "pin":
   if len(rest)!=2{return 2}
   if err:=store.Pin(rest[1]);err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "pair-host":
   if len(rest)!=3{return 2}
   if err:=store.PairHost(rest[1],rest[2]);err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "pair-status":
   if len(rest)!=2{return 2}
   pair,err:=store.ReadPair(rest[1]);if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
   fmt.Printf("host: %s\nglenv: %s\n",pair.HostDeployment,pair.GlenvVersion)
  case "boot-pair-pin":
   if len(rest)!=3{return 2}
   if err:=store.PinPairedHost(rest[1],rest[2]);err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "bind-host":
   if len(rest)!=3{return 2}
   if err:=store.BindHost(rest[1],rest[2]);err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "boot-pin":
   if len(rest)!=3{return 2}
   if err:=store.PinForHost(rest[1],rest[2]);err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "boot-status":
   if len(rest)!=1{return 2}
   binding,err:=store.ReadBinding();if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
   fmt.Printf("host: %s\nglenv: %s\n",binding.HostDeployment,binding.GlenvVersion)
  case "rollback":
   if len(rest)!=1{return 2}
   if err:=store.Rollback();err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
  case "status":
   if len(rest)!=1{return 2}
   current,err:=store.Current();if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
   previous,err:=store.Previous();if err!=nil{fmt.Fprintln(os.Stderr,err);return 1}
   fmt.Printf("store: %s\ncurrent: %s\nprevious: %s\n",filepath.Clean(*storeRoot),current,previous)
  default:fmt.Fprintln(os.Stderr,"unknown release command:",rest[0]);return 2
  }
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
func usage(){fmt.Fprintln(os.Stderr,"usage: glenv <version|inspect|doctor|run|release> [options]")}
