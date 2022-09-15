package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println(`Available commands:
	init:     Initialize new account in current directory
	show:     Show current account information
	export:   Export account public key to file
	friend:   Add a friend using this public key file
	friends:  List all friends
	unfriend: Remove a friend
	follow:   Follow friend posts
	unfollow: Unfollow friend posts
	follows:  List friends you follow
	share:    Share a file with friends following you
	files:    List files you share with your followers
	open:     Open a file shared with you
	delete:   Delete a file you shared previously
	serve:    Open a server to allow followers to sync your content
	sync:     Sync content from a friend`)
		os.Exit(1)
	}

	wd, _ := os.Getwd()

	switch os.Args[1] {
	case "init":
		initCmd := flag.NewFlagSet("init", flag.ExitOnError)
		name := initCmd.String("name", "", "name")
		email := initCmd.String("email", "", "email")
		initCmd.Parse(os.Args[2:])

		passphrase := getPassword()

		fmt.Println("Initializing account...")
		_, err := NewAccount(wd, *name, *email, passphrase)
		raise(err)
		fmt.Println("Done")

	case "show":
		account := getAccount()

		fmt.Println("Name: ", account.Name())
		fmt.Println("Email: ", account.Email())
		fmt.Println("Fingerprint: ", account.Fingerprint())

	case "export":
		exportCmd := flag.NewFlagSet("export", flag.ExitOnError)
		output := exportCmd.String("output", "/dev/stdout", "output")
		exportCmd.Parse(os.Args[2:])

		account := getAccount()

		key, err := account.Export()
		raise(err)

		out, err := os.OpenFile(*output, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
		raise(err)
		defer out.Close()

		fmt.Fprintln(out, string(key))

	case "friend":
		friendCmd := flag.NewFlagSet("friend", flag.ExitOnError)
		key := friendCmd.String("key", "", "path to key file")
		friendCmd.Parse(os.Args[2:])

		account := getAccount()

		keyFile, err := os.OpenFile(*key, os.O_RDONLY, 0600)
		raise(err)
		defer keyFile.Close()

		friend, err := AddFriend(account, keyFile)
		raise(err)

		fmt.Println("Friend added: ", friend.Name(), friend.Email(), friend.Fingerprint())

	case "friends":
		account := getAccount()

		friends, err := ListFriends(account)
		raise(err)

		printKeyRing("", friends)

	case "unfriend":
		unfriendCmd := flag.NewFlagSet("unfriend", flag.ExitOnError)
		fingerprint := unfriendCmd.String("fingerprint", "", "fingerprint of the friend account")
		unfriendCmd.Parse(os.Args[2:])

		account := getAccount()

		friends, err := ListFriends(account)
		raise(err)

		friend := friends.FindByFingerprint(*fingerprint)
		if friend == nil {
			log.Fatal("Can't find friend with fingerprint ", *fingerprint)
		}

		fmt.Println("Removing friend: ", friend.Name(), friend.Email(), "...")
		raise(RemoveFriend(account, friend))
		fmt.Println("Done")

	case "follow":
		followCmd := flag.NewFlagSet("follow", flag.ExitOnError)
		fingerprint := followCmd.String("fingerprint", "", "fingerprint of the friend account")
		followCmd.Parse(os.Args[2:])

		account := getAccount()

		friends, err := ListFriends(account)
		raise(err)

		friend := friends.FindByFingerprint(*fingerprint)
		if friend == nil {
			log.Fatal("Can't find friend with fingerprint ", *fingerprint)
		}

		fmt.Println("Following friend: ", friend.Name(), friend.Email(), "...")
		raise(Follow(account, friend))
		fmt.Println("Done")

	case "unfollow":
		unfollowCmd := flag.NewFlagSet("unfollow", flag.ExitOnError)
		fingerprint := unfollowCmd.String("fingerprint", "", "fingerprint of the friend account")
		unfollowCmd.Parse(os.Args[2:])

		account := getAccount()

		friends, err := ListFriends(account)
		raise(err)

		friend := friends.FindByFingerprint(*fingerprint)
		if friend == nil {
			log.Fatal("Can't find friend with fingerprint ", *fingerprint)
		}

		fmt.Println("Unfollowing friend: ", friend.Name(), friend.Email(), "...")
		raise(Unfollow(account, friend))
		fmt.Println("Done")

	case "follows":
		account := getAccount()

		friends, err := ListFollows(account)
		raise(err)

		for _, f := range friends {
			fmt.Println(f.Name(), f.Email(), f.Fingerprint())
		}

	case "share":
		shareCmd := flag.NewFlagSet("share", flag.ExitOnError)
		file := shareCmd.String("file", "", "file path to share")
		fingerprints := shareCmd.String("fingerprints", "", "comma separated list of fingerprints to share the file with")
		shareCmd.Parse(os.Args[2:])

		account := getAccount()

		f, err := os.Open(*file)
		raise(err)

		allFrields, err := ListFriends(account)
		raise(err)

		fprs := strings.Split(*fingerprints, ",")
		friends := []*Friend{}
		for _, fpr := range fprs {
			f := allFrields.FindByFingerprint(fpr)
			if f == nil {
				raise(fmt.Errorf("Can't find friend %s", fpr))
			}

			friends = append(friends, f)
		}

		name := path.Base(*file)
		_, err = AddFile(account, f, name, friends)
		raise(err)

	case "files":
		filesCmd := flag.NewFlagSet("files", flag.ExitOnError)
		fingerprint := filesCmd.String("fingerprint", "", "fingerprint of the friend account")
		filesCmd.Parse(os.Args[2:])

		account := getAccount()

		if *fingerprint == "" {
			*fingerprint = account.Fingerprint()
		}

		after := time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
		files := ListFiles(account, *fingerprint, after, 0)
		for _, f := range files {
			if f.Deleted(account) {
				continue
			}

			fmt.Println(f.Name())
			rs, err := f.Recipients(account)
			if err != nil {
				fmt.Println("Error: ", err)
				continue
			}

			for _, r := range rs {
				fmt.Println("\t", r.Name(), r.Email(), r.Fingerprint())
			}
		}

	case "open":
		openCmd := flag.NewFlagSet("open", flag.ExitOnError)
		file := openCmd.String("file", "", "file path to open")
		output := openCmd.String("output", "/dev/stdout", "output")
		fingerprint := openCmd.String("fingerprint", "", "fingerprint of the friend account")
		openCmd.Parse(os.Args[2:])

		account := getAccount()

		if *fingerprint == "" {
			*fingerprint = account.Fingerprint()
		}

		f, err := GetFile(account, *fingerprint, *file)
		raise(err)

		r, err := f.Reader(account)
		raise(err)

		out, err := os.OpenFile(*output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		raise(err)
		defer out.Close()

		_, err = io.Copy(out, r)
		raise(err)

	case "delete":
		deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
		file := deleteCmd.String("file", "", "file name to delete")
		deleteCmd.Parse(os.Args[2:])

		account := getAccount()
		f, err := GetFile(account, account.Fingerprint(), *file)
		raise(err)

		raise(RemoveFile(account, f))
		fmt.Println("Deleted", f.Name())

	case "serve":
		account := getAccount()
		server, err := NewServer(account)
		raise(err)

		listener, err := net.Listen("tcp", ":0")
		raise(err)

		port := listener.Addr().(*net.TCPAddr).Port
		fmt.Println("Account: ", account.Name(), account.Fingerprint())
		fmt.Println("Using port:", port)

		server.Serve(listener)

	case "sync":
		syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
		fpr := syncCmd.String("fingerprint", "", "user fingerprint to sync files")
		address := syncCmd.String("address", "", "source address to sync from")
		syncCmd.Parse(os.Args[2:])

		account := getAccount()
		client, err := NewClient(account)
		raise(err)

		// TODO get the latest synced file date
		t := time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC)

		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
		err = DownloadFriend(ctx, account, *address, *fpr, t, client)
		raise(err)

	default:
		fmt.Printf("Command %s is not recognized", os.Args[1])
	}
}

func getAccount() *Account {
	wd, _ := os.Getwd()
	account, err := OpenAccount(wd, getPassword())
	raise(err)

	return account
}

func getPassword() string {
	fmt.Print("Passphrase: ")
	bytepw, err := term.ReadPassword(int(syscall.Stdin))
	raise(err)
	fmt.Println("")

	return string(bytepw)
}

func printKeyRing(p string, r *KeyRing) {
	fmt.Println(r.Name(), ":")
	for _, f := range r.Friends {
		fmt.Println(p+" ", f.Name(), f.Email(), f.Fingerprint())
	}
	for _, k := range r.KeyRings {
		printKeyRing(p+" ", k)
	}
}

func raise(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
