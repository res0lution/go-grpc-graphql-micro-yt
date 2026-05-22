package config

import (
 "fmt"
 "os"
 "strconv"
 "time"

 "github.com/joho/godotenv"
 "github.com/sirupsen/logrus"
)

const (
 defaultConnectTimeout = 15 * time.Second
 defaultSlaUpdateHour  = 2
)

type PoolConfig struct {
 MaxConns         int32
 MinConns         int32
 MaxConnLifetime  time.Duration
 MaxConnIdleTime  time.Duration
 HealthCheckPeriod time.Duration
}

type Inventory struct {
 Host     string
 Port     string
 User     string
 Password string
 DbName   string
 Pool     PoolConfig
}

type Jira struct {
 BaseUrl             string
 Token               string
 ProjectKey          string
 IssueType           string
 PriorityName        string
 PriorityId          string
 EpicPriorityName    string
 EpicPriorityId      string
 StoryPriorityName   string
 StoryPriorityId     string
 SubTaskPriorityName string
 SubTaskPriorityId   string
 EpicParentLinkKey   string
 EpicLinkField       string
}

type AIAssessment struct {
 Space      string
 CSSEpic    string
 AppSecEpic string
 IssueType  string
}

type Api struct {
 TriggerUrl   string
 BasicAuthUser string
 BasicAuthPass string
 MaxRetries    int
 Timeout       time.Duration
}

type App struct {
 LogLevel         string
 GinMode          string
 DuplicateTimeout int
 SlaUpdateHour    int
 SlaRetentionDays int
 SaveDetailedData bool
 ConnectTimeout   time.Duration
}

type Config struct {
 Inventory   Inventory
 Jira        Jira
 AIAssessment AIAssessment
 Feedback    Api
 Notification Api
 App         App
}

func Load(logger *logrus.Entry) (*Config, error) {
 if logger == nil {
  logger = logrus.NewEntry(logrus.StandardLogger())
 }

 if err := godotenv.Load(); err != nil {
  logger.WithField("error", err).Debug(".env file not found")
 }

 cfg := &Config{
  App: App{
   ConnectTimeout: defaultConnectTimeout,
  },
 }

 if err := cfg.loadInventory(); err != nil {
  return nil, fmt.Errorf("inventory config: %w", err)
 }

 cfg.loadJira()
 cfg.loadAIAssessment()
 cfg.loadFeedback()
 cfg.loadNotification()
 cfg.loadApp()

 return cfg, nil
}

func (c *Config) loadInventory() error {
 maxConns := getEnvAsInt("INVENTORY_MAX_CONNS", 10)
 minConns := getEnvAsInt("INVENTORY_MIN_CONNS", 2)
 maxConnLifeTime := getEnvAsInt("INVENTORY_MAX_CONN_LIFETIME_MINUTES", 60)
 maxConnIdleTime := getEnvAsInt("INVENTORY_MAX_CONN_IDLE_TIME_MINUTES", 30)
 healthCheckPeriod := getEnvAsInt("INVENTORY_HEALTH_CHECK_PERIOD_MINUTES", 1)

 c.Inventory = Inventory{
  Host:     getEnv("INVENTORY_HOST", "localhost"),
  Port:     getEnv("INVENTORY_PORT", "5432"),
  User:     getEnv("INVENTORY_USER", ""),
  Password: getEnv("INVENTORY_PASSWORD", ""),
  DbName:   getEnv("INVENTORY_DB_NAME", "inventory"),
  Pool: PoolConfig{
   MaxConns:         int32(maxConns),
   MinConns:         int32(minConns),
   MaxConnLifetime:  time.Duration(maxConnLifeTime) * time.Minute,
   MaxConnIdleTime:  time.Duration(maxConnIdleTime) * time.Minute,
   HealthCheckPeriod: time.Duration(healthCheckPeriod) * time.Minute,
  },
 }

 if c.Inventory.User == "" {
  return fmt.Errorf("INVENTORY_USER is required")
 }

 if c.Inventory.DbName == "" {
  return fmt.Errorf("INVENTORY_DB_NAME is required")
 }

 return nil
}

func (c *Config) loadJira() {
 c.Jira = Jira{
  BaseUrl:             getEnv("JIRA_URL", "http://localhost:8888"),
  Token:               getEnv("JIRA_TOKEN", ""),
  ProjectKey:          getEnv("JIRA_KEY", ""),
  IssueType:           getEnv("JIRA_ISSUE_TYPE", "Task"),
  PriorityName:        getEnv("JIRA_PRIORITY_NAME", ""),
  PriorityId:          getEnv("JIRA_PRIORITY_ID", ""),
  EpicPriorityName:    getEnv("JIRA_EPIC_PRIORITY_NAME", ""),
  EpicPriorityId:      getEnv("JIRA_EPIC_PRIORITY_ID", ""),
  StoryPriorityName:   getEnv("JIRA_STORY_PRIORITY_NAME", ""),
  StoryPriorityId:     getEnv("JIRA_STORY_PRIORITY_ID", ""),
  SubTaskPriorityName: getEnv("JIRA_SUBTASK_PRIORITY_NAME", ""),
  SubTaskPriorityId:   getEnv("JIRA_SUBTASK_PRIORITY_ID", ""),
  EpicParentLinkKey:   getEnv("JIRA_EPIC_PARENT_LINK_KEY", ""),
  EpicLinkField:       getEnv("JIRA_EPIC_LINK_FIELD", ""),
 }
}

func (c *Config) loadAIAssessment() {
	c.AIAssessment = AIAssessment{
	 Space:      getEnv("AI_ASSESSMENT_JIRA_SPACE", "CSS"),
	 CSSEpic:    getEnv("AI_ASSESSMENT_CSS_EPIC", "CSS-1273"),
	 AppSecEpic: getEnv("AI_ASSESSMENT_APPSEC_EPIC", "APPSEC-784"),
	 IssueType:  getEnv("AI_ASSESSMENT_JIRA_ISSUE_TYPE", "Task"),
	}
 }
 
 func (c *Config) loadFeedback() {
	c.Feedback = Api{
	 TriggerUrl:   getEnv("FEEDBACK_API_TRIGGER_URL", "http://localhost:8000"),
	 BasicAuthUser: getEnv("FEEDBACK_API_BASIC_AUTH_USER", ""),
	 BasicAuthPass: getEnv("FEEDBACK_API_BASIC_AUTH_PASS", ""),
	 MaxRetries:    getEnvAsInt("FEEDBACK_API_MAX_RETRIES", 3),
	 Timeout:       time.Duration(getEnvAsInt("FEEDBACK_API_TIMEOUT_SECONDS", 10)) * time.Second,
	}
 }
 
 func (c *Config) loadNotification() {
	c.Notification = Api{
	 TriggerUrl:   getEnv("NOTIFICATION_API_TRIGGER_URL", "http://localhost:8080"),
	 BasicAuthUser: getEnv("NOTIFICATION_API_BASIC_AUTH_USER", ""),
	 BasicAuthPass: getEnv("NOTIFICATION_API_BASIC_AUTH_PASS", ""),
	 MaxRetries:    getEnvAsInt("NOTIFICATION_API_MAX_RETRIES", 3),
	 Timeout:       time.Duration(getEnvAsInt("NOTIFICATION_API_TIMEOUT_SECONDS", 10)) * time.Second,
	}
 }
 
 func (c *Config) loadApp() {
	slaUpdateHour := getEnvAsInt("SLA_UPDATE_HOUR", defaultSlaUpdateHour)
 
	if slaUpdateHour < 0 || slaUpdateHour > 23 {
	 slaUpdateHour = defaultSlaUpdateHour
	}
 
	c.App = App{
	 LogLevel:         getEnv("LOG_LEVEL", "debug"),
	 GinMode:          getEnv("GIN_MODE", "debug"),
	 DuplicateTimeout: getEnvAsInt("DUPLICATE_TIMEOUT_MINUTES", 60),
	 SlaUpdateHour:    slaUpdateHour,
	 SlaRetentionDays: getEnvAsInt("SLA_RETENTION_DAYS", 90),
	 SaveDetailedData: getEnvAsBool("SAVE_DETAILED_DATA", false),
	 ConnectTimeout:   time.Duration(getEnvAsInt("DB_CONNECT_TIMEOUT_SECONDS", 15)) * time.Second,
	}
 }
 
 func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
	 return value
	}
 
	return defaultValue
 }
 
 func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
	 if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	 }
	}
 
	return defaultValue
 }
 
 func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
	 if boolVal, err := strconv.ParseBool(value); err == nil {
		return boolVal
	 }
	}
 
	return defaultValue
 }