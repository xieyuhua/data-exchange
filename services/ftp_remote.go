package services

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"data-exchange/models"

	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// RemoteFileInfo 远程服务器文件信息
type RemoteFileInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
	IsDir   bool   `json:"is_dir"`
}

// resolveRemoteTarget 将账号基础路径与相对名拼接，并做路径穿越防护
// 始终以账号 RemotePath 为根，name 中的 ".." 会被 path.Clean 消除
func resolveRemoteTarget(acc *models.FTPAccount, name string) (string, error) {
	base := strings.TrimRight(acc.RemotePath, "/")
	if base == "" {
		base = "/"
	}
	clean := path.Clean("/" + name) // 规范化，去除 ".." 与多余分隔符
	if clean == "/" || clean == "." || clean == "" {
		return "", fmt.Errorf("非法的文件名称: %s", name)
	}
	return base + clean, nil
}

// ListRemoteFilesResult 远程文件列表（含分页信息）
type ListRemoteFilesResult struct {
	Total int64             `json:"total"`
	List  []RemoteFileInfo  `json:"list"`
}

// ListRemoteFiles 列出远程账号根目录下的文件，支持关键字过滤与分页
// keyword 为空表示不过滤；page/pageSize 做内存分页（目录始终排在文件前）
func (a *App) ListRemoteFiles(acc *models.FTPAccount, keyword string, page, pageSize int) (*ListRemoteFilesResult, error) {
	var all []RemoteFileInfo
	var err error
	switch acc.Protocol {
	case "sftp":
		all, err = a.listSFTPFiles(acc)
	case "ftp":
		all, err = a.listFTPFiles(acc)
	default:
		return nil, fmt.Errorf("不支持的协议: %s", acc.Protocol)
	}
	if err != nil {
		return nil, err
	}

	// 关键字过滤（不区分大小写，匹配文件名）
	if keyword != "" {
		kw := strings.ToLower(keyword)
		filtered := make([]RemoteFileInfo, 0, len(all))
		for _, f := range all {
			if strings.Contains(strings.ToLower(f.Name), kw) {
				filtered = append(filtered, f)
			}
		}
		all = filtered
	}

	// 目录始终排在文件前，目录/文件内部按名称升序
	sortRemoteFiles(all)

	total := int64(len(all))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start > len(all) {
		start = len(all)
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return &ListRemoteFilesResult{Total: total, List: all[start:end]}, nil
}

// sortRemoteFiles 目录优先、名称升序
func sortRemoteFiles(files []RemoteFileInfo) {
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			a, b := files[i], files[j]
			if a.IsDir != b.IsDir {
				if !a.IsDir { // a 是文件、b 是目录，需交换
					files[i], files[j] = files[j], files[i]
				}
				continue
			}
			if a.Name > b.Name {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
}

// DeleteRemoteFile 删除远程文件/目录（name 为相对于 RemotePath 的名称）
func (a *App) DeleteRemoteFile(acc *models.FTPAccount, name string) error {
	target, err := resolveRemoteTarget(acc, name)
	if err != nil {
		return err
	}
	switch acc.Protocol {
	case "sftp":
		return a.deleteSFTP(acc, target)
	case "ftp":
		return a.deleteFTP(acc, target)
	default:
		return fmt.Errorf("不支持的协议: %s", acc.Protocol)
	}
}

// UploadRemoteFile 上传本地文件到远程（remoteName 为相对于 RemotePath 的名称）
func (a *App) UploadRemoteFile(acc *models.FTPAccount, localPath, remoteName string) error {
	target, err := resolveRemoteTarget(acc, remoteName)
	if err != nil {
		return err
	}
	switch acc.Protocol {
	case "sftp":
		return a.uploadSFTPTo(acc, localPath, target)
	case "ftp":
		return a.uploadFTPTo(acc, localPath, target)
	default:
		return fmt.Errorf("不支持的协议: %s", acc.Protocol)
	}
}

// RemoteFileDownload 远程文件下载句柄：流读取器、大小与连接释放函数
type RemoteFileDownload struct {
	Reader io.ReadCloser
	Size   int64
	Close  func() error
}

// DownloadRemoteFile 从远程下载文件（name 为相对于 RemotePath 的名称），返回可读流
// 调用方读取完成后必须调用 Close 释放底层连接
func (a *App) DownloadRemoteFile(acc *models.FTPAccount, name string) (*RemoteFileDownload, error) {
	target, err := resolveRemoteTarget(acc, name)
	if err != nil {
		return nil, err
	}
	switch acc.Protocol {
	case "sftp":
		return a.downloadSFTPFrom(acc, target)
	case "ftp":
		return a.downloadFTPFrom(acc, target)
	default:
		return nil, fmt.Errorf("不支持的协议: %s", acc.Protocol)
	}
}

func (a *App) downloadSFTPFrom(acc *models.FTPAccount, target string) (*RemoteFileDownload, error) {
	client, conn, err := dialSFTP(acc)
	if err != nil {
		return nil, err
	}
	remoteFile, err := client.Open(target)
	if err != nil {
		client.Close()
		conn.Close()
		return nil, fmt.Errorf("打开远程文件失败: %v", err)
	}
	size := int64(0)
	if info, statErr := remoteFile.Stat(); statErr == nil {
		size = info.Size()
	}
	closer := func() error {
		remoteFile.Close()
		client.Close()
		conn.Close()
		return nil
	}
	return &RemoteFileDownload{Reader: remoteFile, Size: size, Close: closer}, nil
}

func (a *App) downloadFTPFrom(acc *models.FTPAccount, target string) (*RemoteFileDownload, error) {
	conn, err := ftp.Dial(fmt.Sprintf("%s:%d", acc.Host, acc.Port), ftp.DialWithTimeout(15*time.Second))
	if err != nil {
		return nil, fmt.Errorf("FTP连接失败: %v", err)
	}
	if err := conn.Login(acc.Username, acc.Password); err != nil {
		conn.Quit()
		return nil, fmt.Errorf("FTP登录失败: %v", err)
	}
	// 直接使用 resolveRemoteTarget 计算出的完整路径，避免 ChangeDir + BaseName 造成的路径错位
	reader, err := conn.Retr(target)
	if err != nil {
		conn.Quit()
		return nil, fmt.Errorf("读取远程文件失败: %v", err)
	}
	size := int64(0)
	if sz, szErr := conn.FileSize(target); szErr == nil {
		size = sz
	}
	closer := func() error {
		reader.Close()
		conn.Quit()
		return nil
	}
	return &RemoteFileDownload{Reader: reader, Size: size, Close: closer}, nil
}

// ==================== SFTP 实现 ====================

func (a *App) listSFTPFiles(acc *models.FTPAccount) ([]RemoteFileInfo, error) {
	client, conn, err := dialSFTP(acc)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer client.Close()

	entries, err := client.ReadDir(acc.RemotePath)
	if err != nil {
		return nil, fmt.Errorf("读取远程目录失败: %v", err)
	}
	res := make([]RemoteFileInfo, 0, len(entries))
	for _, e := range entries {
		res = append(res, RemoteFileInfo{
			Name:    e.Name(),
			Size:    e.Size(),
			ModTime: e.ModTime().Format("2006-01-02 15:04:05"),
			IsDir:   e.IsDir(),
		})
	}
	return res, nil
}

func (a *App) deleteSFTP(acc *models.FTPAccount, target string) error {
	client, conn, err := dialSFTP(acc)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()

	info, err := client.Stat(target)
	if err != nil {
		return fmt.Errorf("文件不存在: %v", err)
	}
	if info.IsDir() {
		if err := client.RemoveDirectory(target); err != nil {
			return fmt.Errorf("删除目录失败: %v", err)
		}
		return nil
	}
	if err := client.Remove(target); err != nil {
		return fmt.Errorf("删除文件失败: %v", err)
	}
	return nil
}

func (a *App) uploadSFTPTo(acc *models.FTPAccount, localPath, target string) error {
	client, conn, err := dialSFTP(acc)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer client.Close()

	dir := path.Dir(target)
	if dir != "" && dir != "/" && dir != "." {
		if err := client.MkdirAll(dir); err != nil {
			return fmt.Errorf("创建远程目录失败: %v", err)
		}
	}

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %v", err)
	}
	defer localFile.Close()

	remoteFile, err := client.Create(target)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %v", err)
	}
	defer remoteFile.Close()

	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("文件上传失败: %v", err)
	}
	return nil
}

func dialSFTP(acc *models.FTPAccount) (*sftp.Client, *ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            acc.Username,
		Auth:            []ssh.AuthMethod{ssh.Password(acc.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", acc.Host, acc.Port), config)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH连接失败: %v", err)
	}
	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("SFTP会话建立失败: %v", err)
	}
	return client, conn, nil
}

// ==================== FTP 实现 ====================

func (a *App) listFTPFiles(acc *models.FTPAccount) ([]RemoteFileInfo, error) {
	conn, err := ftp.Dial(fmt.Sprintf("%s:%d", acc.Host, acc.Port), ftp.DialWithTimeout(15*time.Second))
	if err != nil {
		return nil, fmt.Errorf("FTP连接失败: %v", err)
	}
	defer conn.Quit()
	if err := conn.Login(acc.Username, acc.Password); err != nil {
		return nil, fmt.Errorf("FTP登录失败: %v", err)
	}
	entries, err := conn.List(acc.RemotePath)
	if err != nil {
		return nil, fmt.Errorf("读取远程目录失败: %v", err)
	}
	res := make([]RemoteFileInfo, 0, len(entries))
	for _, e := range entries {
		if e.Name == "." || e.Name == ".." {
			continue
		}
		res = append(res, RemoteFileInfo{
			Name:    e.Name,
			Size:    int64(e.Size),
			ModTime: e.Time.Format("2006-01-02 15:04:05"),
			IsDir:   e.Type == ftp.EntryTypeFolder,
		})
	}
	return res, nil
}

func (a *App) deleteFTP(acc *models.FTPAccount, target string) error {
	conn, err := ftp.Dial(fmt.Sprintf("%s:%d", acc.Host, acc.Port), ftp.DialWithTimeout(15*time.Second))
	if err != nil {
		return fmt.Errorf("FTP连接失败: %v", err)
	}
	defer conn.Quit()
	if err := conn.Login(acc.Username, acc.Password); err != nil {
		return fmt.Errorf("FTP登录失败: %v", err)
	}
	// 直接使用 resolveRemoteTarget 计算出的完整路径，避免 ChangeDir + BaseName 造成的路径错位
	if err := conn.Delete(target); err != nil {
		// 文件删除失败，尝试作为目录删除
		if err2 := conn.RemoveDir(target); err2 != nil {
			return fmt.Errorf("删除失败: %v", err)
		}
	}
	return nil
}

func (a *App) uploadFTPTo(acc *models.FTPAccount, localPath, target string) error {
	conn, err := ftp.Dial(fmt.Sprintf("%s:%d", acc.Host, acc.Port), ftp.DialWithTimeout(15*time.Second))
	if err != nil {
		return fmt.Errorf("FTP连接失败: %v", err)
	}
	defer conn.Quit()
	if err := conn.Login(acc.Username, acc.Password); err != nil {
		return fmt.Errorf("FTP登录失败: %v", err)
	}
	dir := path.Dir(target)
	if dir != "" && dir != "/" && dir != "." {
		// 逐层确保远程目录存在（先尝试切换，失败则创建）
		parts := strings.Split(strings.Trim(dir, "/"), "/")
		cur := ""
		for _, d := range parts {
			if d == "" {
				continue
			}
			cur = cur + "/" + d
			if err := conn.ChangeDir(cur); err != nil {
				if mkErr := conn.MakeDir(cur); mkErr != nil {
					return fmt.Errorf("创建远程目录失败: %v", mkErr)
				}
			}
		}
	}
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %v", err)
	}
	defer localFile.Close()

	if err := conn.Stor(target, localFile); err != nil {
		return fmt.Errorf("FTP上传失败: %v", err)
	}
	return nil
}
