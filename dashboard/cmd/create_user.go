package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

func CreateUserCmd() *cli.Command {
	return &cli.Command{
		Name:  "create-user",
		Usage: "创建用户账号（运维用）",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "email", Aliases: []string{"e"}, Usage: "邮箱（登录账号）", Required: true},
			&cli.StringFlag{Name: "name", Aliases: []string{"n"}, Usage: "用户名称", Required: true},
			&cli.StringFlag{Name: "password", Aliases: []string{"p"}, Usage: "密码", Required: true},
			&cli.StringFlag{Name: "role", Aliases: []string{"r"}, Usage: "角色：super_admin / line_admin / member", Value: "member"},
			&cli.StringFlag{Name: "conf", Aliases: []string{"c"}, Usage: "配置文件", Value: "conf/dashboard-settings.yaml"},
		},
		Action: func(ctx *cli.Context) error {
			email := ctx.String("email")
			name := ctx.String("name")
			password := ctx.String("password")
			role := ctx.String("role")
			confFile := ctx.String("conf")

			if role != "super_admin" && role != "line_admin" && role != "member" {
				return fmt.Errorf("角色必须是 super_admin / line_admin / member")
			}

			db, err := openMySQLFromConfig(confFile)
			if err != nil {
				return fmt.Errorf("连接数据库失败: %v", err)
			}
			defer db.Close()

			userID := generateRandomHex()
			passwordHash := bcryptHash(password)

			_, err = db.Exec(`
				INSERT INTO users (user_id, email, name, password_hash, role, status)
				VALUES (?, ?, ?, ?, ?, 'active')
				ON DUPLICATE KEY UPDATE name = ?, password_hash = ?, role = ?, status = 'active'
			`, userID, email, name, passwordHash, role, name, passwordHash, role)

			if err != nil {
				return fmt.Errorf("创建用户失败: %v", err)
			}

			log.Printf("[OK] 用户创建成功: %s (%s) 角色=%s", name, email, role)
			return nil
		},
	}
}

func openMySQLFromConfig(confFile string) (*sql.DB, error) {
	bs, err := os.ReadFile(confFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var settings struct {
		MySQL struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
			Database string `yaml:"database"`
		} `yaml:"mysql"`
	}

	if err := yaml.Unmarshal(bs, &settings); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true",
		settings.MySQL.User, settings.MySQL.Password,
		settings.MySQL.Host, settings.MySQL.Port, settings.MySQL.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func bcryptHash(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("密码哈希失败: %v", err)
	}
	return string(hash)
}

func generateRandomHex() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
