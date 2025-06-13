package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type UserXML struct {
	Id       int    `xml:"id"`
	Name     string `xml:"first_name"`
	LastName string `xml:"last_name"`
	Age      int    `xml:"age"`
	About    string `xml:"about"`
	Gender   string `xml:"gender"`
}

type SearchResponseXML struct {
	XMLName xml.Name  `xml:"root"`
	Users   []UserXML `xml:"row"`
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	accessToken := r.Header.Get("AccessToken")
	if accessToken == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	xmlFile, err := os.Open("dataset.xml")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errResp := SearchErrorResponse{Error: "error open file"}
		json.NewEncoder(w).Encode(errResp)
		return
	}
	defer xmlFile.Close()

	byteValue, err := io.ReadAll(xmlFile)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errResp := SearchErrorResponse{Error: "error reading file"}
		json.NewEncoder(w).Encode(errResp)
		return
	}

	var UsersXml SearchResponseXML
	if err := xml.Unmarshal(byteValue, &UsersXml); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errResp := SearchErrorResponse{Error: "error parsing XML"}
		json.NewEncoder(w).Encode(errResp)
		return
	}

	searchReq, err := parseSearchParams(r)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if err.Error() == "ErrorBadOrderField" {
			errResp := SearchErrorResponse{Error: "ErrorBadOrderField"}
			json.NewEncoder(w).Encode(errResp)
		} else {
			errResp := SearchErrorResponse{Error: err.Error()}
			json.NewEncoder(w).Encode(errResp)
		}
		return
	}

	filteredUsers := filterUsers(UsersXml.Users, searchReq.Query)

	if searchReq.OrderBy != 0 {
		sort.Slice(filteredUsers, func(i, j int) bool {
			var less bool
			switch searchReq.OrderField {
			case "id":
				less = filteredUsers[i].Id < filteredUsers[j].Id
			case "name":
				less = filteredUsers[i].Name < filteredUsers[j].Name
			case "age":
				less = filteredUsers[i].Age < filteredUsers[j].Age
			}
			if searchReq.OrderBy == -1 {
				return !less
			}
			return less
		})
	}

	users := make([]User, len(filteredUsers))
	for i, u := range filteredUsers {
		users[i] = User{
			Id:     u.Id,
			Name:   strings.TrimSpace(u.Name + " " + u.LastName),
			Age:    u.Age,
			About:  u.About,
			Gender: u.Gender,
		}
	}

	if searchReq.Offset > len(users) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errResp := SearchErrorResponse{Error: "error offset"}
		json.NewEncoder(w).Encode(errResp)
		return
	}

	paginatedUsers := users[searchReq.Offset:]
	if searchReq.Offset+searchReq.Limit < len(users) {
		paginatedUsers = users[searchReq.Offset : searchReq.Offset+searchReq.Limit]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paginatedUsers)
}

func parseSearchParams(r *http.Request) (*SearchRequest, error) {
	query := r.URL.Query()
	req := &SearchRequest{}

	req.Query = query.Get("query")

	orderField := query.Get("order_field")
	if orderField != "name" && orderField != "age" && orderField != "id" && orderField != "" {
		return nil, fmt.Errorf("ErrorBadOrderField")
	}
	if orderField == "" {
		orderField = "name"
	}
	req.OrderField = orderField

	if orderByStr := query.Get("order_by"); orderByStr != "" {
		orderBy, err := strconv.Atoi(orderByStr)
		if err != nil {
			return nil, fmt.Errorf("invalid order_by parameter")
		}
		if orderBy != 1 && orderBy != -1 && orderBy != 0 {
			return nil, fmt.Errorf("order_by must be 1 (asc) or -1 (desc) or 0")
		}
		req.OrderBy = orderBy
	} else {
		req.OrderBy = 0
	}

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil {
		return nil, fmt.Errorf("limit must be int")
	}
	req.Limit = limit

	offset, err := strconv.Atoi(query.Get("offset"))
	if err != nil {
		return nil, fmt.Errorf("offset must be int")
	}
	req.Offset = offset

	return req, nil
}

func filterUsers(users []UserXML, query string) []UserXML {
	if query == "" {
		return users
	}

	var filtered []UserXML
	query = strings.ToLower(query)

	for _, user := range users {
		searchText := strings.ToLower(fmt.Sprintf("%s %s %s",
			user.Name, user.LastName, user.About))

		if strings.Contains(searchText, query) {
			filtered = append(filtered, user)
		}
	}

	return filtered
}

func createTestDataset() error {
	testData := `<?xml version="1.0" encoding="UTF-8"?>
<root>
	<row>
		<id>1</id>
		<first_name>John</first_name>
		<last_name>Doe</last_name>
		<age>25</age>
		<about>Software engineer who loves coding</about>
		<gender>male</gender>
	</row>
	<row>
		<id>2</id>
		<first_name>Jane</first_name>
		<last_name>Smith</last_name>
		<age>30</age>
		<about>Data scientist working with machine learning</about>
		<gender>female</gender>
	</row>
	<row>
		<id>3</id>
		<first_name>Alice</first_name>
		<last_name>Johnson</last_name>
		<age>22</age>
		<about>Product manager in tech company</about>
		<gender>female</gender>
	</row>
</root>`
	return os.WriteFile("dataset.xml", []byte(testData), 0644)
}

// Тестовые серверы для разных ошибок
func ErrorServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func TimeoutServer(w http.ResponseWriter, r *http.Request) {
	time.Sleep(2 * time.Second)
	w.WriteHeader(http.StatusOK)
}

func BadJSONServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(`{"invalid json`))
}

func BadResultJSONServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"invalid": "json"`))
}

func UnknownBadRequestServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	errResp := SearchErrorResponse{Error: "some unknown error"}
	json.NewEncoder(w).Encode(errResp)
}

func TestFindUsers(t *testing.T) {
	err := createTestDataset()
	if err != nil {
		t.Fatalf("Failed to create test dataset: %v", err)
	}
	defer os.Remove("dataset.xml")

	testCases := []struct {
		name          string
		request       SearchRequest
		server        http.HandlerFunc
		accessToken   string
		expectedError string
		expectedUsers int
		expectedNext  bool
	}{
		{
			name: "Успешный поиск всех пользователей",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        SearchServer,
			accessToken:   "valid_token",
			expectedUsers: 3,
			expectedNext:  false,
		},
		{
			name: "Поиск с фильтрацией",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "engineer",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        SearchServer,
			accessToken:   "valid_token",
			expectedUsers: 1,
			expectedNext:  false,
		},
		{
			name: "Сортировка по возрасту по убыванию",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "",
				OrderField: "age",
				OrderBy:    OrderByAsc, // -1
			},
			server:        SearchServer,
			accessToken:   "valid_token",
			expectedUsers: 3,
			expectedNext:  false,
		},
		{
			name: "Пагинация с NextPage",
			request: SearchRequest{
				Limit:      2,
				Offset:     0,
				Query:      "",
				OrderField: "id",
				OrderBy:    OrderByDesc, // 1
			},
			server:        SearchServer,
			accessToken:   "valid_token",
			expectedUsers: 2,
			expectedNext:  true,
		},
		{
			name: "Limit меньше 0",
			request: SearchRequest{
				Limit:      -1,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        SearchServer,
			accessToken:   "valid_token",
			expectedError: "limit must be > 0",
		},
		{
			name: "Offset меньше 0",
			request: SearchRequest{
				Limit:      10,
				Offset:     -1,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        SearchServer,
			accessToken:   "valid_token",
			expectedError: "offset must be > 0",
		},
		{
			name: "Limit больше 25",
			request: SearchRequest{
				Limit:      30,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        SearchServer,
			accessToken:   "valid_token",
			expectedUsers: 3,
			expectedNext:  false,
		},
		{
			name: "Неавторизованный запрос",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        SearchServer,
			accessToken:   "",
			expectedError: "Bad AccessToken",
		},
		{
			name: "Неверное поле сортировки",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "",
				OrderField: "invalid_field",
				OrderBy:    0,
			},
			server:        SearchServer,
			accessToken:   "valid_token",
			expectedError: "OrderFeld invalid_field invalid",
		},
		{
			name: "Ошибка сервера",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        ErrorServer,
			accessToken:   "valid_token",
			expectedError: "SearchServer fatal error",
		},
		{
			name: "Timeout",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        TimeoutServer,
			accessToken:   "valid_token",
			expectedError: "timeout for",
		},
		{
			name: "Неверный JSON в ошибке",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        BadJSONServer,
			accessToken:   "valid_token",
			expectedError: "cant unpack error json:",
		},
		{
			name: "Неверный JSON в результате",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        BadResultJSONServer,
			accessToken:   "valid_token",
			expectedError: "cant unpack result json:",
		},
		{
			name: "Неизвестная ошибка BadRequest",
			request: SearchRequest{
				Limit:      10,
				Offset:     0,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        UnknownBadRequestServer,
			accessToken:   "valid_token",
			expectedError: "unknown bad request error:",
		},
		{
			name: "Слишком большой offset",
			request: SearchRequest{
				Limit:      10,
				Offset:     100,
				Query:      "",
				OrderField: "name",
				OrderBy:    0,
			},
			server:        SearchServer,
			accessToken:   "valid_token",
			expectedError: "unknown bad request error: error offset",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.server)
			defer server.Close()

			searchClient := &SearchClient{
				AccessToken: tc.accessToken,
				URL:         server.URL,
			}

			result, err := searchClient.FindUsers(tc.request)

			if tc.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tc.expectedError)
					return
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tc.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result.Users) != tc.expectedUsers {
				t.Errorf("Expected %d users, got %d", tc.expectedUsers, len(result.Users))
			}

			if result.NextPage != tc.expectedNext {
				t.Errorf("Expected NextPage %v, got %v", tc.expectedNext, result.NextPage)
			}

			// Проверяем правильность сортировки
			if tc.request.OrderBy == OrderByAsc && tc.request.OrderField == "age" && len(result.Users) > 1 {
				if result.Users[0].Age < result.Users[1].Age {
					t.Error("Users should be sorted by age in descending order")
				}
			}
		})
	}
}

// Тест для получения реальной unknown error через закрытое соединение
func TestUnknownNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close() // Закрываем сервер для получения connection refused

	searchClient := &SearchClient{
		AccessToken: "token",
		URL:         url,
	}

	request := SearchRequest{
		Limit:      10,
		Offset:     0,
		Query:      "",
		OrderField: "name",
		OrderBy:    0,
	}

	_, err := searchClient.FindUsers(request)
	if err == nil {
		t.Error("Expected connection error, got nil")
	}

	if !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("Expected 'unknown error', got: %s", err.Error())
	}
}
