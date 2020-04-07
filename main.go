package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/deckarep/golang-set"
	"github.com/fsnotify/fsnotify"
	"github.com/otiai10/copy"

	"okentaro/fsw-copy-go/lib"
)

var watcher *fsnotify.Watcher
var srcDir string
var destDir []string


func init() {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// 監視ディレクトリ追加
	srcDir = os.Args[1]
	if err := watcher.Add(srcDir); err != nil {
		log.Fatal(err)
	}
	log.Println("added watching directory", srcDir)
	if err := filepath.Walk(srcDir, func(p string, info os.FileInfo, err error) error {
		if info.IsDir()  {
			if err := watcher.Add(p); err != nil {
				log.Fatal(err)
			}
			log.Println("added watching directory", p)
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	// コピー先ディレクトリ追加
	destDir = os.Args[2:]
	if len(destDir) == 0 {
		log.Fatal("no dest dir")
	}
	log.Println("dest directories", destDir)

	// ディレクトリ一旦全部コピー
	for _, directory := range destDir {
		if err := lib.RemoveAll(directory); err != nil {
			log.Fatal(err)
		}
		if err := copy.Copy(srcDir, directory); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	done := make(chan bool)
	queue := make(chan fsnotify.Event)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Println("watcher stopped")
					done <- true
					return
				}
				log.Println("event:", event)
				queue <- event
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Println("watcher stopped")
					done <- true
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	go dispatcher(5, queue, copyFiles(srcDir, destDir...))
	<-done
}

// dispatcher ch への書き込みが途切れてから interval 秒後に run 実行
func dispatcher(interval int, changed chan fsnotify.Event, run func([]string, []string) error) {
	intervalDuration := time.Duration(interval) * time.Second
	timer := time.NewTimer(0)
	changeFiles := mapset.NewSet()
	removeFiles := mapset.NewSet()

	for {
		select {
		case event, ok := <-changed:
			// ファイル変更検知
			log.Println("file change detected")
			if !ok {
				log.Println("dispatcher channel closed")
				return
			}

			timer.Stop()
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				if changeFiles.Contains(event.Name) {
					changeFiles.Remove(event.Name)
				} else {
					removeFiles.Add(event.Name)
				}
			} else {
				changeFiles.Add(event.Name)
			}
			timer.Reset(intervalDuration)
		case <-timer.C:
			// 一定時間後に run 実行
			if err := run(lib.ToStringArray(changeFiles), lib.ToStringArray(removeFiles)); err != nil {
				log.Fatal("dispatch func failed ", err)
			}
			changeFiles.Clear()
			removeFiles.Clear()
		}
	}
}

// copy files to destDir
func copyFiles(srcBaseDir string, destDir ...string) func([]string, []string) error {
	return func(changeFiles, removeFiles []string) error {
		wg := &sync.WaitGroup{}
		count := (len(changeFiles) + len(removeFiles)) * len(destDir)
		wg.Add(count)

		// 変更ファイルコピー
		for _, file := range changeFiles {
			if lib.IsDirOrInvalidFile(file, srcBaseDir) {
				lib.DoneWaitGroups(wg, len(destDir))
				continue
			}

			info, err := os.Stat(file)
			if err != nil {
				return err
			}
			for _, destBaseDir := range destDir {
				go func() {
					defer wg.Done()

					destFilePath := strings.Replace(file, srcBaseDir, destBaseDir, 1)
					if info.IsDir() {
						// ディレクトリ作成
						if err := os.MkdirAll(destFilePath, info.Mode()); err != nil {
							log.Println(err)
							fmt.Printf("failed to create directory %s\n", destFilePath)
						} else {
							fmt.Printf("directory %s created\n", destFilePath)
						}
					} else {
						// ファイルコピー
						if err := lib.CopyFile(file, destFilePath); err != nil {
							log.Println(err)
							fmt.Printf("failed to copy file from %s to %s\n", file, destFilePath)
						} else {
							fmt.Printf("file copied from %s to %s\n", file, destFilePath)
						}
					}
				}()
			}

			// ディレクトリの場合は watcher に追加
			if info.IsDir() {
				if err := watcher.Add(file); err != nil {
					// 監視に追加できなかったので終了
					return err
				}
				log.Println("added watching directory", file)
			}
		}

		// 削除ファイルの削除
		for _, file := range removeFiles {
			for _, destBaseDir := range destDir {
				destFilePath := strings.Replace(file, srcBaseDir, destBaseDir, 1)
				if lib.IsDirOrInvalidFile(destFilePath, destBaseDir) {
					wg.Done()
					continue
				}

				go func() {
					defer wg.Done()

					info, err := os.Stat(destFilePath)
					if err != nil {
						log.Println(err)
						return
					}
					if info.IsDir() {
						// ディレクトリ以下全削除
						if err := lib.RemoveAll(destFilePath); err != nil {
							log.Println(err)
							fmt.Printf("failed to remove directory %s\n", destFilePath)
						} else {
							fmt.Printf("directory %s removed\n", destFilePath)
						}
					} else {
						// ファイル削除
						if err := os.Remove(destFilePath); err != nil {
							log.Println(err)
							fmt.Printf("failed to remove file %s\n", destFilePath)
						} else {
							fmt.Printf("file %s removed\n", destFilePath)
						}
					}

				}()
			}
		}

		// 終了
		wg.Wait()
		return nil
	}
}




