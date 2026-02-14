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

	. "github.com/mau-network/mau"
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
		if err := initCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse init flags: %v", err)
		}

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
		if err := exportCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse export flags: %v", err)
		}

		account := getAccount()

		out, err := os.OpenFile(*output, os.O_CREATE|os.O_RDWR|os.O_TRUNC, FilePerm)
		raise(err)
		defer out.Close()

		err = account.Export(out)
		raise(err)

	case "friend":
		friendCmd := flag.NewFlagSet("friend", flag.ExitOnError)
		key := friendCmd.String("key", "", "path to key file")
		if err := friendCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse friend flags: %v", err)
		}

		account := getAccount()

		keyFile, err := os.OpenFile(*key, os.O_RDONLY, 0600)
		raise(err)
		defer keyFile.Close()

		friend, err := account.AddFriend(keyFile)
		raise(err)

		fmt.Println("Friend added: ", friend.Name(), friend.Email(), friend.Fingerprint())

	case "friends":
		account := getAccount()

		friends, err := account.ListFriends()
		raise(err)

		printKeyring("", friends)

	case "unfriend":
		unfriendCmd := flag.NewFlagSet("unfriend", flag.ExitOnError)
		fingerprint := unfriendCmd.String("fingerprint", "", "fingerprint of the friend account")
		if err := unfriendCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse unfriend flags: %v", err)
		}

		account := getAccount()

		friends, err := account.ListFriends()
		raise(err)

		fpr, err := FingerprintFromString(*fingerprint)
		raise(err)

		friend := friends.FindByFingerprint(fpr)
		if friend == nil {
			log.Fatal("Can't find friend with fingerprint ", *fingerprint)
		}

		fmt.Println("Removing friend: ", friend.Name(), friend.Email(), "...")
		raise(account.RemoveFriend(friend))
		fmt.Println("Done")

	case "follow":
		followCmd := flag.NewFlagSet("follow", flag.ExitOnError)
		fingerprint := followCmd.String("fingerprint", "", "fingerprint of the friend account")
		if err := followCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse follow flags: %v", err)
		}

		account := getAccount()

		friends, err := account.ListFriends()
		raise(err)

		fpr, err := FingerprintFromString(*fingerprint)
		raise(err)

		friend := friends.FindByFingerprint(fpr)
		if friend == nil {
			log.Fatal("Can't find friend with fingerprint ", *fingerprint)
		}

		fmt.Println("Following friend: ", friend.Name(), friend.Email(), "...")
		raise(account.Follow(friend))
		fmt.Println("Done")

	case "unfollow":
		unfollowCmd := flag.NewFlagSet("unfollow", flag.ExitOnError)
		fingerprint := unfollowCmd.String("fingerprint", "", "fingerprint of the friend account")
		if err := unfollowCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse unfollow flags: %v", err)
		}

		account := getAccount()

		friends, err := account.ListFriends()
		raise(err)

		fpr, err := FingerprintFromString(*fingerprint)
		raise(err)

		friend := friends.FindByFingerprint(fpr)
		if friend == nil {
			log.Fatal("Can't find friend with fingerprint ", *fingerprint)
		}

		fmt.Println("Unfollowing friend: ", friend.Name(), friend.Email(), "...")
		raise(account.Unfollow(friend))
		fmt.Println("Done")

	case "follows":
		account := getAccount()

		friends, err := account.ListFollows()
		raise(err)

		for _, f := range friends {
			fmt.Println(f.Name(), f.Email(), f.Fingerprint())
		}

	case "share":
		shareCmd := flag.NewFlagSet("share", flag.ExitOnError)
		file := shareCmd.String("file", "", "file path to share")
		fingerprints := shareCmd.String("fingerprints", "", "comma separated list of fingerprints to share the file with")
		if err := shareCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse share flags: %v", err)
		}

		account := getAccount()

		f, err := os.Open(*file)
		raise(err)

		allFrields, err := account.ListFriends()
		raise(err)

		fprs := strings.Split(*fingerprints, ",")
		friends := []*Friend{}
		for _, fprStr := range fprs {
			fpr, err := FingerprintFromString(fprStr)
			if err != nil {
				raise(fmt.Errorf("Can't parse %s as fingerprint", fprStr))
			}

			f := allFrields.FindByFingerprint(fpr)
			if f == nil {
				raise(fmt.Errorf("Can't find friend %s", fprStr))
			}

			friends = append(friends, f)
		}

		name := path.Base(*file)
		_, err = account.AddFile(f, name, friends)
		raise(err)

	case "files":
		filesCmd := flag.NewFlagSet("files", flag.ExitOnError)
		fingerprint := filesCmd.String("fingerprint", "", "fingerprint of the friend account")
		if err := filesCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse files flags: %v", err)
		}

		account := getAccount()
		var fpr Fingerprint

		if *fingerprint == "" {
			fpr = account.Fingerprint()
		} else {
			var err error
			fpr, err = FingerprintFromString(*fingerprint)
			raise(err)
		}

		after := time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
		files := account.ListFiles(fpr, after, 0)
		for _, f := range files {
			if f.Deleted() {
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
		if err := openCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse open flags: %v", err)
		}

		account := getAccount()
		var fpr Fingerprint

		if *fingerprint == "" {
			fpr = account.Fingerprint()
		} else {
			var err error
			fpr, err = FingerprintFromString(*fingerprint)
			raise(err)
		}

		f, err := account.GetFile(fpr, *file)
		raise(err)

		r, err := f.Reader(account)
		raise(err)

		out, err := os.OpenFile(*output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePerm)
		raise(err)
		defer out.Close()

		_, err = io.Copy(out, r)
		raise(err)

	case "delete":
		deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
		file := deleteCmd.String("file", "", "file name to delete")
		if err := deleteCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse delete flags: %v", err)
		}

		account := getAccount()
		f, err := account.GetFile(account.Fingerprint(), *file)
		raise(err)

		raise(account.RemoveFile(f))
		fmt.Println("Deleted", f.Name())

	case "serve":
		account := getAccount()
		server, err := account.Server(nil)
		raise(err)

		listener, err := net.Listen("tcp", ":0")
		raise(err)

		port := listener.Addr().(*net.TCPAddr).Port
		fmt.Println("Account: ", account.Name(), account.Fingerprint())
		fmt.Println("Using port:", port)

		if err := server.Serve(listener, ""); err != nil {
			log.Fatalf("Server error: %v", err)
		}

	case "sync":
		syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
		fprStr := syncCmd.String("fingerprint", "", "user fingerprint to sync files")
		address := syncCmd.String("address", "", "source address to sync from")
		if err := syncCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse sync flags: %v", err)
		}

		account := getAccount()
		var fpr Fingerprint
		fpr, err := FingerprintFromString(*fprStr)
		raise(err)

		client, err := account.Client(fpr, nil)
		raise(err)

		// TODO get the latest synced file date
		t := time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		resolvers := []FingerprintResolver{LocalFriendAddress}
		if len(*address) > 0 {
			resolvers = append(resolvers, StaticAddress(*address))
		}
		err = client.DownloadFriend(ctx, fpr, t, resolvers)
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

func printKeyring(p string, r *Keyring) {
	fmt.Println(r.Name(), ":")
	for _, f := range r.Friends {
		fmt.Println(p+" ", f.Name(), f.Email(), f.Fingerprint())
	}
	for _, k := range r.SubKeyrings {
		printKeyring(p+" ", k)
	}
}

func raise(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
