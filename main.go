package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
)

const (
	cros_sdk_path     = "/usr/local/google/home/whalechang/depot_tools/cros_sdk"
	user_root         = "/usr/local/google/home/whalechang"
	mirror_path       = "chromiumos_repo/mirror"
	main_repo_path    = "chromiumos"
	internal_manifest = "https://chrome-internal.googlesource.com/chromeos/manifest-internal"
	repo_url          = "https://chromium.googlesource.com/external/repo.git"
)

func run_command(cmd *exec.Cmd) error {
	fmt.Println("runing ", cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Stderr.Write(out)
		fmt.Println("command failed:", cmd)
		return err
	}
	return nil
}

func get_repo_path(suffix string) (string, error) {
	if suffix != "debug" && suffix != "common" && suffix != "stable" && suffix != "project" {
		return "", errors.New("no such path")
	}
	return fmt.Sprintf("%s/chromiumos_repo/chromiumos.%s", user_root, suffix), nil
}

func get_main_repo_path() string {
	return fmt.Sprintf("%s/%s", user_root, main_repo_path)
}

func get_mirror_repo_path() string {
	return fmt.Sprintf("%s/%s", user_root, mirror_path)
}

func cros_run_command(cros_cmd []string) error {
	cmd := exec.Command(cros_sdk_path)
	for _, arg := range cros_cmd {
		cmd.Args = append(cmd.Args, arg)
	}
	cmd.Dir = get_main_repo_path()
	if err := run_command(cmd); err != nil {
		return err
	}
	return nil
}

func point_to_main_repo(repo_path string) error {
	var err error
	fmt.Printf("removing %s ...\n", get_main_repo_path())
	os.Remove(get_main_repo_path())
	fmt.Printf("create soft link %s to %s ...\n", repo_path, get_main_repo_path())
	cmd := exec.Command("ln", "-s", repo_path, get_main_repo_path())
	if err = run_command(cmd); err != nil {
		return err
	}
	return nil
}

func sync_mirror() error {
	fmt.Println("syncing mirror ...")
	cmd := exec.Command("repo", "sync", "-j128")
	cmd.Dir = get_mirror_repo_path()
	if err := run_command(cmd); err != nil {
		return err
	}
	return nil
}

func recreate_repo(repo_path string) error {
	fmt.Printf("removing %s ...\n", repo_path)
	now := time.Now() // current local time
	sec := now.Unix()
	cmd := exec.Command("mv", repo_path, fmt.Sprintf("/tmp/%s.%d", filepath.Base(repo_path), sec))
	if err := run_command(cmd); err != nil {
		fmt.Println(err)
	}
	fmt.Printf("mkdir %s ...\n", repo_path)
	if err := os.Mkdir(repo_path, 0755); err != nil {
		return err
	}
	fmt.Printf("repo init %s ...\n", repo_path)
	cmd = exec.Command(
		"repo",
		"init",
		"--reference", get_mirror_repo_path(),
		"-u", internal_manifest,
		"--repo-url", repo_url,
		"-b", "main",
	)
	cmd.Dir = repo_path
	if err := run_command(cmd); err != nil {
		return err
	}
	return nil
}

func repo_sync(repo_path string) error {
	fmt.Printf("repo syncing %s ...\n", repo_path)
	cmd := exec.Command(
		"repo",
		"sync",
		"-j128",
	)
	cmd.Dir = repo_path
	if err := run_command(cmd); err != nil {
		return err
	}
	return nil
}

func repo_sync_remote(repo_path string) error {
	fmt.Printf("repo syncing %s ...\n", repo_path)
	cmd := exec.Command(
		"repo",
		"sync",
		"-n",
		"-j128",
	)
	cmd.Dir = repo_path
	if err := run_command(cmd); err != nil {
		return err
	}
	return nil
}

func repo_forall(repo_path string, cmds []string) error {
	fmt.Printf("repo forall %s ...\n", repo_path)
	cmd := exec.Command(
		"repo",
		"forall",
		"-c",
	)
	for _, c := range cmds {
		cmd.Args = append(cmd.Args, c)
	}
	cmd.Dir = repo_path
	if err := run_command(cmd); err != nil {
		return err
	}
	return nil

}

func build_pacakges(boards []string, pacakges []string) error {
	var err error
	for _, board := range boards {
		for _, pacakge := range pacakges {
			setup_board := []string{"setup_board", "--force", "--board", board}
			if err = cros_run_command(setup_board); err != nil {
				return err
			}
			build_package := []string{"build_packages", "--autosetgov", fmt.Sprintf("--board=%s", board), pacakge}
			if err = cros_run_command(build_package); err != nil {
				return err
			}
			emerge := []string{fmt.Sprintf("emerge-%s", board), pacakge}
			if err = cros_run_command(emerge); err != nil {
				return err
			}
		}
	}
	return nil
}

func debug(repo_path string, sync bool, build bool, boards []string, packages []string) error {
	var err error
	if sync {
		if err = sync_mirror(); err != nil {
			return err
		}
		if err = recreate_repo(repo_path); err != nil {
			return err
		}
		if err = repo_sync(repo_path); err != nil {
			return err
		}
	}
	if build {
		if err = build_pacakges(boards, packages); err != nil {
			return err
		}
	}
	return nil
}

func common(repo_path string, sync bool, build bool, boards []string, packages []string) error {
	var err error
	if sync {
		if err = sync_mirror(); err != nil {
			return err
		}
		if err = repo_sync_remote(repo_path); err != nil {
			return err
		}
	}
	if build {
		if err = build_pacakges(boards, packages); err != nil {
			return err
		}
	}
	return nil
}

func stable(repo_path string, sync bool, build bool, boards []string, packages []string) error {
	var err error
	if sync {
		if err = sync_mirror(); err != nil {
			return err
		}
	}
	if build {
		if err = build_pacakges(boards, packages); err != nil {
			return err
		}
	}
	return nil
}

func run(opts options) {
	var repo_path string
	var err error
	if repo_path, err = get_repo_path(opts.usage); err != nil {
		log.Fatalf("%s", err)
	}
	if err = point_to_main_repo(repo_path); err != nil {
		log.Fatalf("%s", err)
	}
	switch usage := opts.usage; usage {
	case "debug":
		if err = debug(repo_path, opts.sync, opts.build, opts.boards, opts.packages); err != nil {
			log.Fatalf("%s", err)
		}
	case "common":
		if err = common(repo_path, opts.sync, opts.build, opts.boards, opts.packages); err != nil {
			log.Fatalf("%s", err)
		}
	case "stable":
		if err = stable(repo_path, opts.sync, opts.build, opts.boards, opts.packages); err != nil {
			log.Fatalf("%s", err)
		}
	case "project":
		fmt.Println("project will not build and sync")
	default:
		log.Fatalf("%s", "bug: no such usage")
	}
}

type options struct {
	usage    string
	sync     bool
	build    bool
	boards   []string
	packages []string
}

func main() {
	var opts options
	var boards string
	var packages string
	kingpin.Arg("usage", "repo target: debug common stable. e.x. --usage=debug").Required().StringVar(&opts.usage)
	kingpin.Flag("boards", "board to build. e.x. --board brya,corsola").StringVar(&boards)
	kingpin.Flag("packages", "packages to build. e.x. --packages adhd,floss").StringVar(&packages)
	kingpin.Flag("sync", "sync the repo").Default("false").BoolVar(&opts.sync)
	kingpin.Flag("build", "build the package for boards").Default("false").BoolVar(&opts.build)
	kingpin.Parse()

	opts.boards = strings.Split(boards, ",")
	opts.packages = strings.Split(packages, ",")
	run(opts)
}
