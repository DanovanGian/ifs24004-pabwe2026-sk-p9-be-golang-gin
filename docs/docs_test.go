package docs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSwaggerJSON(t *testing.T) {
	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(GetSwaggerJSON("localhost:8081", "http"), &spec))

	paths := spec["paths"].(map[string]interface{})
	assert.Contains(t, paths, "/api/auth/login")
	assert.Contains(t, paths, "/api/posts/{id}")
	assert.Len(t, paths["/api/auth/profile"], 2)

	server := spec["servers"].([]interface{})[0].(map[string]interface{})
	assert.Equal(t, "http://localhost:8081", server["url"])
}
