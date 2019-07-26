// Copyright © 2015-2017 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package upgrade

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	. "github.com/platinasystems/go"
	"github.com/platinasystems/go/internal/url"
)

const (
	MaxImgs = 3
	Goes    = 0
	Kern    = 1
	Core    = 2
	GoesBin = "/usr/bin/goes"
)

type IMGINFO struct {
	Name   string
	Build  string
	User   string
	Size   string
	Tag    string
	Commit string
	Chksum string
}

func printImageInfo() (err error) {
	var imgInfo [MaxImgs]IMGINFO

	if imgInfo[Goes], err = getGoesInfo(); err != nil {
		return err
	}
	if imgInfo[Kern], err = getKernelInfo(); err != nil {
		return err
	}
	if imgInfo[Core], err = getCorebootInfo(); err != nil {
		return err
	}

	fmt.Println("")
	fmt.Print("Currently running:\n")
	for i, _ := range imgInfo {
		prn("    Name   : ", imgInfo[i].Name)
		prn("    Build  : ", imgInfo[i].Build)
		prn("    User   : ", imgInfo[i].User)
		prn("    Size   : ", imgInfo[i].Size)
		prn("    Version: ", imgInfo[i].Tag)
		prn("    Commit : ", imgInfo[i].Commit)
		fmt.Println("")
	}
	return nil
}

func prn(d string, s string) {
	if s != "" {
		fmt.Println(d, s)
	}
}

func getGoesInfo() (im IMGINFO, err error) {
	im.Name = GoesName
	im.Build = getGoesVal("generated.on")
	im.User = getGoesVal("generated.by")
	fi, err := os.Stat(GoesBin)
	if err != nil {
		return im, err
	}
	im.Size = fmt.Sprintf("%d", fi.Size())
	im.Tag = getGoesVal("tag")
	im.Commit = getGoesVal("version")
	return im, nil
}

func getKernelInfo() (im IMGINFO, err error) {
	im.Name, _ = getKernelVal("-s")
	im.Build, _ = getKernelVal("-v")
	im.User = ""
	im.Tag, _ = getKernelVal("-r")
	im.Commit = ""
	im.Size = ""
	fi, _ := os.Stat("/boot/vmlinuz-" + im.Tag)
	if err == nil {
		im.Size = fmt.Sprintf("%d", fi.Size())
	}
	return im, nil
}

func getCorebootInfo() (im IMGINFO, err error) {
	im.Name = CorebootName
	_, err = exec.Command("/usr/local/sbin/flashrom", "-p",
		"internal:boardmismatch=force", "-l",
		"/usr/local/share/flashrom/layouts/platina-mk1.xml",
		"-i", "bios", "-r", "cb.rom", "-A", "-V").Output()
	if err != nil {
		return im, err
	}
	a, err := ioutil.ReadFile("cb.rom")
	if err != nil {
		return im, err
	}
	temp := strings.Split(string(a), "\n")
	for _, j := range temp {
		if strings.Contains(j, "COREBOOT_VERSION ") {
			x := strings.Split(j, " ")
			im.Commit = strings.Replace(x[2], `"`, "", 2)
		}
		if strings.Contains(j, "COREBOOT_BUILD ") {
			x := strings.SplitAfterN(j, " ", 3)
			im.Build = strings.Replace(x[2], `"`, "", 2)
		}
	}
	im.User = ""
	fi, err := os.Stat("cb.rom")
	if err != nil {
		return im, err
	}
	im.Size = fmt.Sprintf("%d", fi.Size())
	im.Tag = ""

	return im, nil
}

func upgradeGoes(s string, v string, t bool, f bool) error {
	fmt.Printf("Update Goes\n")
	if !f {
		g := getGoesVer()
		gr, err := getSrvGoesVer(s, v, t)
		if err != nil {
			return err
		}
		fmt.Printf("    Goes version currently:  %s\n", g)
		fmt.Printf("    Goes version on server:  %s\n", gr)
		if g == gr {
			fmt.Print("    Versions match, skipping Goes upgrade\n\n")
			return nil
		}
		if len(g) == 0 || len(gr) == 0 {
			fmt.Print("    No tag found, aborting Goes upgrade\n\n")
			return nil
		}
	}

	if err := installGoes(s, v, t); err != nil {
		return err
	}
	return nil
}

func upgradeKernel(s string, v string, t bool, f bool) error {
	fmt.Printf("Update Kernel\n")
	kr, fn, err := getSrvKernelVer(s, v, t)
	if err != nil {
		return err
	}
	if !f {
		k, err := getKernelVer()
		if err != nil {
			return err
		}
		fmt.Printf("    Kernel version currently:  %s\n", k)
		fmt.Printf("    Kernel version on server:  %s\n", kr)
		if k == kr {
			fmt.Print("    Versions match, skipping Kernel upgrade\n\n")
			return nil
		}
	}

	if err := installKernel(s, v, t, fn); err != nil {
		return err
	}
	return nil
}

func upgradeCoreboot(s string, v string, t bool, f bool) error {
	fmt.Printf("Update Coreboot\n")
	if !f {
		c, err := getCorebootVer()
		if err != nil {
			return err
		}
		cr, err := getSrvCorebootVer(s, v, t)
		if err != nil {
			return err
		}
		fmt.Printf("    Coreboot version currently:  %s\n", c)
		fmt.Printf("    Coreboot version on server:  %s\n", cr)
		if c == cr {
			fmt.Print("    Versions match, skipping Coreboot upgrade\n\n")
		}
	}

	if err := installCoreboot(s, v, t); err != nil {
		return err
	}
	return nil
}

func getGoesVer() (v string) {
	ar := "tag"
	maps := []map[string]string{Package}
	if Packages != nil {
		maps = append(maps, Packages()...)
	}
	for _, m := range maps {
		if ip, found := m["importpath"]; found {
			k := strings.TrimLeft(ar, "-")
			if val, found := m[k]; found {
				if strings.Contains(ip, "/go") {
					v = strings.Replace(val, "'", "", 1)
				}
			}
		}
	}
	return v
}

func getGoesVal(ar string) (v string) {
	maps := []map[string]string{Package}
	if Packages != nil {
		maps = append(maps, Packages()...)
	}
	for _, m := range maps {
		if ip, found := m["importpath"]; found {
			k := strings.TrimLeft(ar, "-")
			if val, found := m[k]; found {
				if strings.Contains(ip, "/go") {
					v = strings.Replace(val, "'", "", 1)
				}
			}
		}
	}
	return v
}

func getKernelVer() (string, error) {
	u, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(u)), nil
}

func getKernelVal(ar string) (string, error) {
	u, err := exec.Command("uname", ar).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(u)), nil
}

func getCorebootVer() (string, error) { //TODO
	return "no_tag", nil
}

func getSrvGoesVer(s string, v string, t bool) (string, error) {
	fn := GoesName

	n, err := getFile(s, v, t, fn)
	if err != nil {
		return "", fmt.Errorf("Error downloading: %v", err)
	}
	if n < 1000 {
		return "", fmt.Errorf("Error file too small: %v", err)
	}
	x := ""
	a, err := ioutil.ReadFile(fn)
	if err != nil {
		return "", err
	}
	as := string(a)
	ref := regexp.MustCompile("v([0-9])[.]([0-9])-([0-9]+)-g([0-9a-fA-F]+)")
	x = ref.FindString(as)
	if len(x) == 0 {
		ree := regexp.MustCompile("v([0-9])[.]([0-9])")
		x = ree.FindString(as)
	}
	rmFile(fn)
	return x, nil
}

func getSrvKernelVer(s string, v string, t bool) (string, string, error) {
	fn := KernelName
	n, err := getFile(s, v, t, fn)
	if err != nil {
		return "", "", fmt.Errorf("Error downloading: %v", err)
	}
	if n < 10 {
		return "", "", fmt.Errorf("Error file too small: %v", err)
	}
	a, err := ioutil.ReadFile(fn)
	if err != nil {
		return "", "", err
	}
	u := strings.Split(string(a), "\n")
	return strings.TrimSpace(u[0]), strings.TrimSpace(u[1]), nil
}

func getSrvCorebootVer(s string, v string, t bool) (string, error) { //TODO
	return "no_tag", nil
}

func installGoes(s string, v string, t bool) error {
	fn := GoesInstaller
	n, err := getFile(s, v, t, fn)
	if err != nil {
		return fmt.Errorf("    Error downloading: %v", err)
	}
	if n < 1000 {
		return fmt.Errorf("    Error file too small: %v", err)
	}

	Install_flag = true
	return nil
}

func installKernel(s string, v string, t bool, fn string) error {
	n, err := getFile(s, v, t, fn)
	if err != nil {
		return fmt.Errorf("    Error downloading: %v", err)
	}
	if n < 1000 {
		return fmt.Errorf("    Error file too small: %v", err)
	}

	_, err = exec.Command("dpkg", "-i", fn).Output()
	if err != nil {
		return err
	}

	err = cleanupBootDir(fn)
	if err != nil {
		return err
	}

	_, err = exec.Command("update-grub").Output()
	if err != nil {
		return err
	}

	Reboot_flag = true
	return nil
}

func installCoreboot(s string, v string, t bool) error {
	_, err := exec.Command("/usr/local/sbin/flashrom", "-p",
		"internal:boardmismatch=force", "-l",
		"/usr/local/share/flashrom/layouts/platina-mk1.xml",
		"-i", "bios", "-w", "coreboot.rom", "-A", "-V").Output()
	if err != nil {
		return err
	}
	Reboot_flag = true
	return nil
}

func cleanupBootDir(fn string) error {
	ref := regexp.MustCompile("([0-9]+)[.]([0-9]+)[.]([0-9]+)")
	x := ref.FindString(fn)
	if len(x) == 0 {
		return fmt.Errorf("Error: version number not found. %v", fn)
	}
	files, _ := ioutil.ReadDir("/boot")
	for _, f := range files {
		if !f.IsDir() {
			if !strings.Contains(f.Name(), x) {
				err := rmFile("/boot/" + f.Name())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func getFile(s string, v string, t bool, fn string) (int, error) {
	rmFile(fn)
	urls := "http://" + s + "/" + v + "/" + fn
	if t {
		urls = "tftp://" + s + "/" + v + "/" + fn
	}
	r, err := url.Open(urls)
	if err != nil {
		return 0, err
	}
	f, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, DfltMod)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return 0, err
	}
	syscall.Fsync(int(os.Stdout.Fd()))
	return int(n), nil
}

func rmFile(f string) error {
	if _, err := os.Stat(f); err != nil {
		return err
	}
	if err := os.Remove(f); err != nil {
		return err
	}
	return nil
}

func reboot() error {
	fmt.Print("\nWILL REBOOT in 1 minute... Please login again\n")
	u, err := exec.Command("shutdown", "-r", "+1").Output()
	fmt.Println(u)
	if err != nil {
		return err
	}
	return nil
}

func activateGoes() error {
	fmt.Print("\nACTIVATING GOES, WILL EXIT... type reset, goes\n")
	cmd := exec.Command("./" + GoesInstaller)
	cmd.Start()
	return nil
}
