package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/badger/cmd"
	"github.com/dgraph-io/ristretto/z"
	"github.com/dustin/go-humanize"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"go.opencensus.io/zpages"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"runtime"
	"sort"
	"strings"
	"time"
)

// BadgerStore 实现 redis.Storer 接口
type BadgerStore struct {
	db   *badger.DB
	user *User
}

// Set 设置键值对
func (s *BadgerStore) Set(key, value []byte) error {
	if !s.user.HasPermission(WritePermission) {
		return errors.New("user does not have permission to write")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// Get 获取键对应的值
func (s *BadgerStore) Get(key []byte) (int, error) {
	if !s.user.HasPermission(ReadPermission) {
		return 0, errors.New("user does not have permission to read")
	}
	var val []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return 0, err
	}
	return len(val), nil
}

// Delete 删除键值对
func (s *BadgerStore) Delete(key []byte) error {
	if !s.user.HasPermission(DeletePermission) {
		return errors.New("user does not have permission to delete")
	}
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

type User struct {
	Username    string
	Permissions map[string]bool
}

// Permission 枚举类型
type Permission int

func (p Permission) String() string {
	switch p {
	case ReadPermission:
		return "read"
	case WritePermission:
		return "write"
	case DeletePermission:
		return "delete"
	default:
		return ""
	}
}

// 定义 Permission 的枚举值
const (
	ReadPermission Permission = iota
	WritePermission
	DeletePermission
)
const (
	redisServerAddress = "localhost:6379"
)

func (u *User) HasPermission(p Permission) bool {
	return u.Permissions[string(p)]
}
func (s *BadgerStore) checkPermission(ctx context.Context, p Permission) error {
	// 从上下文获取用户信息
	user, ok := ctx.Value("user").(*User)
	if !ok {
		return errors.New("user not found in context")
	}

	// 检查用户是否有权限
	if !user.HasPermission(p) {
		return errors.New("user does not have permission to perform this action")
	}

	return nil
}

func main() {
	go func() {
		for i := 8080; i < 9080; i++ {
			fmt.Printf("Listening for /debug HTTP requests at port: %d\n", i)
			if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", i), nil); err != nil {
				fmt.Println("Port busy. Trying another one...")
				continue

			}
		}
	}()
	zpages.Handle(nil, "/z")
	runtime.SetBlockProfileRate(100)
	runtime.GOMAXPROCS(128)

	out := z.CallocNoRef(1, "Badger.Main")
	fmt.Printf("jemalloc enabled: %v\n", len(out) > 0)
	z.StatsPrint()
	z.Free(out)

	cmd.Execute()
	fmt.Printf("Num Allocated Bytes at program end: %s\n",
		humanize.IBytes(uint64(z.NumAllocBytes())))
	if z.NumAllocBytes() > 0 {
		fmt.Println(z.Leaks())
	}

	//使用新的MyDatabase结构体实例化一个数据库对象，并使用新增的Set、Get、Delete方法
	//各一分钟后启动
	//time.Sleep(time.Minute) // Time to do some poking around.
	db, err := badger.Open(badger.DefaultOptions("path/to/database"))
	if err != nil {
		log.Fatal("Error opening Badger database:", err)
	}

	defer func(db *badger.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	myDB := NewMyDatabase(db)

	key := []byte("key")
	value := []byte("value")

	err = myDB.Set(key, value)
	if err != nil {
		log.Fatal("Error setting value:", err)
	}

	val, err := myDB.Get(key)
	if err != nil {
		log.Fatal("Error getting value:", err)
	}
	log.Println("Value:", string(val))

	err = myDB.Delete(key)
	if err != nil {
		log.Fatal("Error deleting key:", err)
	}

	//兼容 Redis 常见数据结构
	//对Redis数据结构的映射
	// 打开Badger数据库
	db, err = badger.Open(badger.DefaultOptions("my_db"))
	if err != nil {
		fmt.Println("Failed to open Badger database:", err)
		return
	}
	defer func(db *badger.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	//字符串（string）数据结构，可以将键值对的键定义为字符串类型的键，值定义为字符串的值。
	// 存储字符串数据结构
	err = db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("my_key"), []byte("my_value"))
		return err
	})

	// 读取字符串数据结构
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("my_key"))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			fmt.Printf("Value: %s\n", val)
			return nil
		})
		return err
	})

	// 存储列表数据结构
	err = db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("my_list_key"), []byte("value1,value2,value3"))
		return err
	})

	// 读取列表数据结构
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("my_list_key"))
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			values := strings.Split(string(val), ",")
			fmt.Printf("Values: %v\n", values)
			return nil
		})
		return err
	})

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}
	//
	//兼容列表（list）数据结构分布
	// 添加列表项到列表中
	err = db.Update(func(txn *badger.Txn) error {
		key := []byte("my_list_key")
		items := []string{"item1", "item2", "item3"}
		value := []byte(strings.Join(items, ","))

		err := txn.Set(key, value)
		return err
	})

	// 读取列表数据结构
	err = db.View(func(txn *badger.Txn) error {
		key := []byte("my_list_key")
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			values := strings.Split(string(val), ",")
			fmt.Printf("List Items: %v\n", values)
			return nil
		})
		return err
	})

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}
	//
	//
	// 定义结构体表示哈希表
	type Hashmap struct {
		Data map[string]string // 存储键值对映射
	}

	// 创建一个哈希表对象并添加键值对
	hashmap := Hashmap{
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}
	//
	// 将哈希表对象序列化为JSON字符串
	value, err = json.Marshal(hashmap)
	if err != nil {
		fmt.Println("Failed to marshal hashmap to JSON:", err)
		return
	}
	//兼容哈希映射（hashmap）
	// 存储哈希表数据结构
	err = db.Update(func(txn *badger.Txn) error {
		key := []byte("my_hashmap_key")
		err := txn.Set(key, value)
		return err
	})

	// 读取哈希表数据结构并解码JSON
	err = db.View(func(txn *badger.Txn) error {
		key := []byte("my_hashmap_key")
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			var decoded Hashmap
			err := json.Unmarshal(val, &decoded)
			if err != nil {
				return err
			}
			fmt.Printf("Hashmap Data: %v\n", decoded.Data)
			return nil
		})
		return err
	})

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}
	//
	//set数据结构
	// 添加一些元素到集合
	setItems := []string{"item1", "item2", "item3"}

	// 存储集合数据结构
	err = db.Update(func(txn *badger.Txn) error {
		key := []byte("my_set_key")
		value := []byte(strings.Join(setItems, ","))
		err := txn.Set(key, value)
		return err
	})

	// 读取集合数据结构
	err = db.View(func(txn *badger.Txn) error {
		key := []byte("my_set_key")
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			values := strings.Split(string(val), ",")
			fmt.Printf("Set Items: %v\n", values)
			return nil
		})
		return err
	})

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}
	//
	//兼容sortedmap数据结构

	type SortedMap struct {
		Keys   []int
		Values []string
	}
	// 创建一个有序映射
	sortedmap := SortedMap{
		Keys:   []int{1, 3, 2},
		Values: []string{"value1", "value3", "value2"},
	}

	// 将有序映射序列化为JSON字符串
	value, err = json.Marshal(sortedmap)
	if err != nil {
		fmt.Println("Failed to marshal sortedmap to JSON:", err)
		return
	}

	// 存储有序映射数据结构
	err = db.Update(func(txn *badger.Txn) error {
		key := []byte("my_sortedmap_key")
		err := txn.Set(key, value)
		return err
	})

	// 读取有序映射数据结构并解码JSON
	err = db.View(func(txn *badger.Txn) error {
		key := []byte("my_sortedmap_key")
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		err = item.Value(func(val []byte) error {
			var decoded SortedMap
			err := json.Unmarshal(val, &decoded)
			if err != nil {
				return err
			}

			// 根据键排序映射
			sortedKeys := make([]int, len(decoded.Keys))
			copy(sortedKeys, decoded.Keys)
			sort.Ints(sortedKeys)

			// 通过有序键打印值
			for _, key := range sortedKeys {
				index := sort.SearchInts(decoded.Keys, key)
				fmt.Printf("Key: %d, Value: %s\n", key, decoded.Values[index])
			}
			return nil
		})
		return err
	})

	if err != nil {
		fmt.Println("An error occurred:", err)
		return
	}

	//Redis客户端兼容性
	// 创建或打开Badger数据库
	db, err = badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *badger.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	// 创建一个BadgerStore
	store := &BadgerStore{db: db}

	// 启动Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer func(client *redis.Client) {
		err := client.Close()
		if err != nil {

		}
	}(client)

	// 注册Badger的Store到Redis的客户端
	client.WrapProcess(func(old func([]redis.Cmder) error) func([]redis.Cmder) error {
		return func(cmds []redis.Cmder) error {
			ctx := context.Background()
			for _, cmd := range cmds {
				err := store.processCommands(ctx, cmd)
				if err != nil {
					return err
				}
			}
			return nil
		}
	})
	// 关闭第一个 Redis 客户端
	err = client.Close()
	if err != nil {
		log.Println("Failed to close Redis client:", err)
	}
	// 启动Redis客户端，并配置它连接到Badger代理服务器
	client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer func(client *redis.Client) {
		err := client.Close()
		if err != nil {
			log.Println("Failed to close Redis client:", err)
		}
	}(client)
	// 使用BadgerStore处理Redis命令
	client.WrapProcess(func(old func(cmd []redis.Cmder) error) func(cmd []redis.Cmder) error {
		return func(cmds []redis.Cmder) error {
			ctx := context.Background()
			// 处理多个命令
			for _, cmd := range cmds {
				if err := store.processCommands(ctx, cmd); err != nil {
					// 处理错误
					return err
				}
			}
			return nil
		}
	})

	// 交互式命令行
	fmt.Println("准备链接Ready to interact with Badger using redis-cli...")

	for i := 0; i < 3; i++ {
		_, err = client.Ping(context.Background()).Result()
		if err != nil {
			log.Printf("Failed to ping redis server: %v", err)
			time.Sleep(time.Second * 2) // 等待2秒后重试
			continue
		}
		break // 连接成功，退出循环
	}
	user := &User{
		Username: "admin",
		Permissions: map[string]bool{
			ReadPermission.String():   true,
			WritePermission.String():  true,
			DeletePermission.String(): true,
		},
	}

	store = &BadgerStore{
		db:   db,
		user: user,
	}

	//tcp
	listener, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatalf("Error listening: %v", err)
	}
	defer listener.Close()

	// 使用BadgerStore处理Redis命令
	store = &BadgerStore{db: db}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Error accepting connection: %v\\n", err)
				continue
			}
			go handleConnection(conn, store)
		}
	}()
	// 建立与 Redis 服务器的 TCP 连接
	conn, err := net.Dial("tcp", redisServerAddress)
	if err != nil {
		fmt.Printf("Failed to connect to Redis server: %v\\n", err)
		return
	}
	defer conn.Close()

	// 保持程序运行
	for {
		// 这里可以放置一些日志输出，或者进行一些周期性的检查
		fmt.Println("程序正在运行...")
		time.Sleep(1 * time.Minute) // 每秒打印一次，以减少对控制台的输出频率
	}

}
func handleConnection(conn net.Conn, store *BadgerStore) {
	var user *User
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		cmd := scanner.Text()
		log.Println("Received command:", cmd)

		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			log.Println("Invalid command received")
			continue
		}

		switch strings.ToUpper(parts[0]) {
		case "SET":
			// 处理 SET 命令
			if len(parts) < 3 {
				conn.Write([]byte("SET command requires at least 2 arguments\\n"))
				continue
			}
			// 使用 key 变量
			// 使用 value 变量
			// 调用 store.Set 方法将键值对存储起来
			// store.Set(context.TODO(), key, value)
			conn.Write([]byte("OK\\n"))
		case "GET":
			// 处理 GET 命令
			if len(parts) < 2 {
				conn.Write([]byte("GET command requires at least 1 argument\\n"))
				continue
			}
			// 使用 key 变量
			// 调用 store.Get 方法获取指定键的值
			// val, _ := store.Get(context.TODO(), key)
			// conn.Write([]byte(val + "\\n"))
			conn.Write([]byte("Value\\n"))
		default:
			conn.Write([]byte("Unsupported command\\n"))
		}
		_ = context.WithValue(context.Background(), "user", user)

	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from client: %v", err)
	}
}

func (s *BadgerStore) processCommands(ctx context.Context, cmd redis.Cmder) error {
	_ = ctx.Value("user").(*User)

	switch cmd.Name() {
	case "SET":
		args := cmd.Args()
		if len(args) != 3 {
			return redis.Nil
		}
		key := args[1].(string)
		value := args[2].(string)
		if err := s.checkPermission(ctx, WritePermission); err != nil {
			return err
		}
		return s.db.Update(func(txn *badger.Txn) error {
			return txn.Set([]byte(key), []byte(value))
		})
	case "GET":
		args := cmd.Args()
		if len(args) != 2 {
			return redis.Nil
		}
		key := args[1].(string)
		if err := s.checkPermission(ctx, ReadPermission); err != nil {
			return err
		}
		var val []byte
		err := s.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(key))
			if err != nil {
				return err
			}
			val, err = item.ValueCopy(nil)
			return err
		})
		if err != nil {
			return err
		}
		if cmd, ok := cmd.(*redis.Cmd); ok {
			cmd.SetVal(string(val))
		} else {
			return errors.New("cmd is not a *redis.Cmd")
		}
	default:
		return redis.Nil
	}
	return nil
}
