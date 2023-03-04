package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/wechatgpt/wechatbot/config"
)

type STMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Choice struct {
	Index        int       `json:"index"`
	Message      STMessage `json:"message"`
	FinishReason string    `json:"finish_reason"`
}

// ChatGPTResponseBody 请求体
type ChatGPTResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Choices []Choice               `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
}

type ChatGPTErrorBody struct {
	Error map[string]interface{} `json:"error"`
}

// ChatGPTRequestBody 响应体
type ChatGPTRequestBody struct {
	Model    string      `json:"model"`
	Messages []STMessage `json:"messages"`
}

// Completions https://api.openai.com/v1/completions
// nodejs example
// const { Configuration, OpenAIApi } = require("openai");
//
//	 const configuration = new Configuration({
//	   apiKey: process.env.OPENAI_API_KEY,
//	 });
//	 const openai = new OpenAIApi(configuration);
//
//	 const response = await openai.createCompletion({
//	   model: "text-davinci-003",
//	   prompt: "I am a highly intelligent question answering bot. If you ask me a question that is rooted in truth, I will give you the answer. If you ask me a question that is nonsense, trickery, or has no clear answer, I will respond with \"Unknown\".\n\nQ: What is human life expectancy in the United States?\nA: Human life expectancy in the United States is 78 years.\n\nQ: Who was president of the United States in 1955?\nA: Dwight D. Eisenhower was president of the United States in 1955.\n\nQ: Which party did he belong to?\nA: He belonged to the Republican Party.\n\nQ: What is the square root of banana?\nA: Unknown\n\nQ: How does a telescope work?\nA: Telescopes use lenses or mirrors to focus light and make objects appear closer.\n\nQ: Where were the 1992 Olympics held?\nA: The 1992 Olympics were held in Barcelona, Spain.\n\nQ: How many squigs are in a bonk?\nA: Unknown\n\nQ: Where is the Valley of Kings?\nA:",
//	   temperature: 0,
//	   max_tokens: 100,
//	   top_p: 1,
//	   frequency_penalty: 0.0,
//	   presence_penalty: 0.0,
//	   stop: ["\n"],
//	});
//
// Completions sendMsg
var messages []STMessage

func Completions(msg string) (*string, error) {
	apiKey := config.GetOpenAiApiKey()
	if apiKey == nil {
		return nil, errors.New("未配置apiKey")
	}
	//清空指令
	if strings.ToLower(msg) == "/clear" {
		messages = make([]STMessage, 0)
		strOK := "已清空上下文"
		return &strOK, nil
	}

	messages = append(messages, STMessage{
		Role:    "user",
		Content: msg,
	})

	requestBody := ChatGPTRequestBody{
		Model:    "gpt-3.5-turbo",
		Messages: messages,
	}
	requestData, err := json.Marshal(requestBody)

	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Printf("request openai json string : %v", string(requestData))
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestData))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *apiKey))
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(response.Body)

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	gptResponseBody := &ChatGPTResponseBody{}
	log.Println(string(body))
	err = json.Unmarshal(body, gptResponseBody)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var reply string
	if len(gptResponseBody.Choices) > 0 {
		for _, v := range gptResponseBody.Choices {
			message := v.Message
			messages = append(messages, message)
			reply = strings.TrimSpace(message.Content)
			if reply == "" {
				reply = "【API返回了空内容。如果多次出现请清空上下文】"
			}
			break
		}
	}

	gptErrorBody := &ChatGPTErrorBody{}
	err = json.Unmarshal(body, gptErrorBody)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if len(reply) == 0 {
		reply = gptErrorBody.Error["message"].(string)
	}

	log.Printf("gpt response full text: %s \n", reply)
	result := strings.TrimSpace(reply)
	return &result, nil
}
