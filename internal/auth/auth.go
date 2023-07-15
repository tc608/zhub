package auth

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID       int    `yaml:"id"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Status   string `yaml:"status"`

	Groups []string `yaml:"groups"`
	Read   []string `yaml:"reads"`
	Write  []string `yaml:"writes"`

	authCache map[string]string // zcore:userid = rw|w|r|-
	lock      sync.RWMutex
}
type Group struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Read        []string `yaml:"reads"`
	Write       []string `yaml:"writes"`
}
type Token struct {
	ID         int       `yaml:"id"`
	UserID     int       `yaml:"user_id"`
	Token      string    `yaml:"token"`
	Expiration time.Time `yaml:"expiration"`
	Status     string    `yaml:"status"`
	Channels   []string  `yaml:"channels"`
}

type Channel struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Public      bool   `yaml:"public"`
}

type Config struct {
	Users    []*User   `yaml:"users"`
	Groups   []Group   `yaml:"groups"`
	Tokens   []Token   `yaml:"tokens"`
	Channels []Channel `yaml:"channels"`
}

type PermissionManager struct {
	config     Config
	userMap    map[int]*User
	groupMap   map[string]Group
	tokenMap   map[string]Token
	channelMap map[string]Channel
	lock       sync.RWMutex
}

func (p *PermissionManager) Init() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	// Load YAML configuration from file
	data, err := os.ReadFile("./auth.yml")
	if err != nil {
		return err
	}
	// Unmarshal YAML into AuthConfig struct
	err = yaml.Unmarshal(data, &p.config)

	if err != nil {
		return err
	}

	// Build user map
	p.userMap = make(map[int]*User)
	for _, user := range p.config.Users {
		p.userMap[user.ID] = user
	}

	// Build group map
	p.groupMap = make(map[string]Group)
	for _, group := range p.config.Groups {
		p.groupMap[group.Name] = group
	}

	// Build token map
	p.tokenMap = make(map[string]Token)
	for _, token := range p.config.Tokens {
		p.tokenMap[token.Token] = token
	}

	// Build channel map
	p.channelMap = make(map[string]Channel)
	for _, channel := range p.config.Channels {
		p.channelMap[channel.Name] = channel
	}

	// clean cache
	for _, user := range p.userMap {
		user.authCache = make(map[string]string)
	}

	//p.updatePermissions()
	return nil
}

func (p *PermissionManager) Reload() error {
	// Reload the configuration by calling the Init() method again
	err := p.Init()
	if err != nil {
		return err
	}
	return nil
}

func (p *PermissionManager) GetUserIdByToken(token string) (int, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	t, found := p.tokenMap[token]
	if !found || !t.isValid() {
		return 0, fmt.Errorf("invalid or expired token")
	}

	user, found := p.userMap[t.UserID]
	if !found || user.Status != "active" {
		return 0, fmt.Errorf("invalid or expired token")
	}

	return user.ID, nil
}

// AuthCheck cate = r|w
func (p *PermissionManager) AuthCheck(userid int, topic, cate string) bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	user := p.userMap[userid]

	str := user.authCache[topic]
	if str != "" && strings.Contains(str, cate) {
		return true
	}
	c := p.channelMap[topic]
	if c.Public {
		return true
	}

	// collect all auth expression
	exps := make(map[string]bool)
	if cate == "r" {
		for _, exp := range user.Read {
			exps[exp] = true
		}
		for _, g := range user.Groups {
			group := p.groupMap[g]
			for _, exp := range group.Read {
				exps[exp] = true
			}
		}
	} else if cate == "w" {
		for _, exp := range user.Write {
			exps[exp] = true
		}
		for _, g := range user.Groups {
			group := p.groupMap[g]
			for _, exp := range group.Write {
				exps[exp] = true
			}
		}
	}

	matchFound := false
	for exp, _ := range exps {
		matchFound, _ = regexp.MatchString(exp, topic)
		if matchFound {
			break
		}
	}

	if matchFound {
		user.lock.Lock()
		defer user.lock.Unlock()

		if user.authCache == nil {
			user.authCache = make(map[string]string)
		}
		user.authCache[topic] = cate + str
		return true
	}

	return false
}

func (p *PermissionManager) IsAdmin(userId int) bool {
	user := p.userMap[userId]
	return user != nil && user.Username == "admin"
}

// AdminToken server's admin using
func (p *PermissionManager) AdminToken() (string, error) {
	var userId int
	for _, user := range p.userMap {
		if user.Username == "admin" {
			userId = user.ID
		}
	}

	for _, token := range p.tokenMap {
		if token.UserID == userId && token.isValid() {
			return token.Token, nil
		}
	}

	return "", fmt.Errorf("-Error: can't found valid admin token")
}

// isValid check token's valid
func (t *Token) isValid() bool {
	return t.Expiration.After(time.Now()) && t.Status == "active"
}
