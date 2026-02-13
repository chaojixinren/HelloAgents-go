package core

import (
	"os"
	"strconv"
)

// QdrantConfig Qdrant 向量数据库配置。
type QdrantConfig struct {
	// 连接配置
	URL    string
	APIKey string

	// 集合配置
	CollectionName string
	VectorSize     int
	Distance       string

	// 连接配置
	Timeout int
}

// DefaultQdrantConfig 返回默认 Qdrant 配置。
func DefaultQdrantConfig() *QdrantConfig {
	return &QdrantConfig{
		CollectionName: "hello_agents_vectors",
		VectorSize:     384,
		Distance:       "cosine",
		Timeout:        30,
	}
}

// QdrantConfigFromEnv 从环境变量创建 Qdrant 配置。
func QdrantConfigFromEnv() *QdrantConfig {
	c := DefaultQdrantConfig()
	c.URL = os.Getenv("QDRANT_URL")
	c.APIKey = os.Getenv("QDRANT_API_KEY")

	if v := os.Getenv("QDRANT_COLLECTION"); v != "" {
		c.CollectionName = v
	}
	if v := os.Getenv("QDRANT_VECTOR_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.VectorSize = n
		}
	}
	if v := os.Getenv("QDRANT_DISTANCE"); v != "" {
		c.Distance = v
	}
	if v := os.Getenv("QDRANT_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.Timeout = n
		}
	}
	return c
}

// ToMap 转换为 map。
func (c *QdrantConfig) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	if c.URL != "" {
		m["url"] = c.URL
	}
	if c.APIKey != "" {
		m["api_key"] = c.APIKey
	}
	m["collection_name"] = c.CollectionName
	m["vector_size"] = c.VectorSize
	m["distance"] = c.Distance
	m["timeout"] = c.Timeout
	return m
}

// Neo4jConfig Neo4j 图数据库配置。
type Neo4jConfig struct {
	// 连接配置
	URI      string
	Username string
	Password string
	Database string

	// 连接池配置
	MaxConnectionLifetime       int
	MaxConnectionPoolSize       int
	ConnectionAcquisitionTimeout int
}

// DefaultNeo4jConfig 返回默认 Neo4j 配置。
func DefaultNeo4jConfig() *Neo4jConfig {
	return &Neo4jConfig{
		URI:                         "bolt://localhost:7687",
		Username:                    "neo4j",
		Password:                    "hello-agents-password",
		Database:                    "neo4j",
		MaxConnectionLifetime:       3600,
		MaxConnectionPoolSize:       50,
		ConnectionAcquisitionTimeout: 60,
	}
}

// Neo4jConfigFromEnv 从环境变量创建 Neo4j 配置。
func Neo4jConfigFromEnv() *Neo4jConfig {
	c := DefaultNeo4jConfig()

	if v := os.Getenv("NEO4J_URI"); v != "" {
		c.URI = v
	}
	if v := os.Getenv("NEO4J_USERNAME"); v != "" {
		c.Username = v
	}
	if v := os.Getenv("NEO4J_PASSWORD"); v != "" {
		c.Password = v
	}
	if v := os.Getenv("NEO4J_DATABASE"); v != "" {
		c.Database = v
	}
	if v := os.Getenv("NEO4J_MAX_CONNECTION_LIFETIME"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxConnectionLifetime = n
		}
	}
	if v := os.Getenv("NEO4J_MAX_CONNECTION_POOL_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.MaxConnectionPoolSize = n
		}
	}
	if v := os.Getenv("NEO4J_CONNECTION_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.ConnectionAcquisitionTimeout = n
		}
	}
	return c
}

// ToMap 转换为 map。
func (c *Neo4jConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"uri":                          c.URI,
		"username":                     c.Username,
		"password":                     c.Password,
		"database":                     c.Database,
		"max_connection_lifetime":      c.MaxConnectionLifetime,
		"max_connection_pool_size":     c.MaxConnectionPoolSize,
		"connection_acquisition_timeout": c.ConnectionAcquisitionTimeout,
	}
}

// DatabaseConfig 数据库配置管理器。
type DatabaseConfig struct {
	Qdrant *QdrantConfig
	Neo4j  *Neo4jConfig
}

// DefaultDatabaseConfig 返回默认数据库配置。
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Qdrant: DefaultQdrantConfig(),
		Neo4j:  DefaultNeo4jConfig(),
	}
}

// DatabaseConfigFromEnv 从环境变量创建数据库配置。
func DatabaseConfigFromEnv() *DatabaseConfig {
	return &DatabaseConfig{
		Qdrant: QdrantConfigFromEnv(),
		Neo4j:  Neo4jConfigFromEnv(),
	}
}

// GetQdrantConfig 获取 Qdrant 配置 map。
func (c *DatabaseConfig) GetQdrantConfig() map[string]interface{} {
	if c.Qdrant == nil {
		return nil
	}
	return c.Qdrant.ToMap()
}

// GetNeo4jConfig 获取 Neo4j 配置 map。
func (c *DatabaseConfig) GetNeo4jConfig() map[string]interface{} {
	if c.Neo4j == nil {
		return nil
	}
	return c.Neo4j.ToMap()
}

// 全局配置实例
var dbConfig = DatabaseConfigFromEnv()

// GetDatabaseConfig 获取数据库配置。
func GetDatabaseConfig() *DatabaseConfig {
	return dbConfig
}

// UpdateDatabaseConfig 更新数据库配置。
func UpdateDatabaseConfig(qdrant *QdrantConfig, neo4j *Neo4jConfig) {
	if qdrant != nil {
		dbConfig.Qdrant = qdrant
	}
	if neo4j != nil {
		dbConfig.Neo4j = neo4j
	}
}
