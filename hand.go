package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

var errAccountAlreadyExist = errors.New("account already exist")
var errAccountsDontExist = errors.New("accounts not found")
var errSuchAccountDontExist = errors.New("accounts not found")

// ^ создаём новую ошибку с данным текстом, чтобы использовать в будущем

func NewAccount(name string, value int) *Account { // метод инициализации для структуры Account
	return &Account{ // создаёт новый экземпляр этой структуры с двумя переданными значениями, и возвращает указатель на этот экземпляр
		Name:  name,
		Value: value,
	}
}

type Account struct { // структура данных для создания аккаунта
	Name  string
	Value int
}

type CreateAccountRequest struct { // json-овский тип данных для создания запроса
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type Handler struct { // фигня для обработки рот ебал получаемых данных и блять фиксирования изменений в них
	accounts map[string]*Account
	rwMutex  *sync.RWMutex
}

func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) { // обработка http запроса и ответа
	var names []string // локально храним данные
	var values []int
	switch req.Method { // смотрим используемый метод и от него выбираем свои действия
	case http.MethodPost:
		var createAccountRequest CreateAccountRequest                                   // создаём форму для будущей загрузки на сервак
		if err := json.NewDecoder(req.Body).Decode(&createAccountRequest); err != nil { // обработка ошибки, чтобы серв не лёг
			// смотрим, не наебали ли нас в http
			resp.WriteHeader(http.StatusInternalServerError)
			_, _ = resp.Write([]byte("parse create account request failed"))

			return
		}

		if err := h.createAccount(createAccountRequest.Name, createAccountRequest.Value); err != nil {
			if errors.Is(err, errAccountAlreadyExist) { // смотрим, не наебали ли нас с аккаунтом (существует)
				resp.WriteHeader(http.StatusConflict)
				_, _ = resp.Write([]byte("account already exist"))

				return
			}

			resp.WriteHeader(http.StatusInternalServerError) // смотрим, не наебали ли нас с аккаунтом (что-то другое)
			_, _ = resp.Write([]byte("create account failed"))

			return
		}

		resp.WriteHeader(http.StatusCreated) // если доп условий не встретилось подтверждаем успешность и возвращаем новый акк
		_, _ = resp.Write([]byte("create account success"))

		return

	case http.MethodGet:

		if err := h.GetAccountNames(); errors.Is(err, errAccountsDontExist) {
			resp.WriteHeader(http.StatusNotFound)
			_, _ = resp.Write([]byte("accounts not found"))
			return
		}
		resp.WriteHeader(http.StatusFound) // если доп условий не встретилось подтверждаем успешность
		_, _ = resp.Write([]byte("accounts have been successfully founded"))

		for _, value := range h.accounts { // запрашиваем данные формата имя аккаунта значение
			names = append(names, value.Name)
			values = append(values, value.Value)
		}
		fmt.Println("get: names, values", names, values)

	case http.MethodDelete:
		for _, value := range h.accounts { // запрашиваем данные формата имя аккаунта значение
			names = append(names, value.Name)
			values = append(values, value.Value)
		} // костыль. Без данного цикла не получается взять записанные в методГете данные
		fmt.Println("delete, values:", values)
		if err := h.DeleteAccounts(names); err != nil {
			resp.WriteHeader(http.StatusNotFound)
			_, _ = resp.Write([]byte("accounts not found"))
			return
		}
		resp.WriteHeader(http.StatusFound) // если доп условий не встретилось подтверждаем успешность
		_, _ = resp.Write([]byte("accounts have been successfully deleted"))

	case http.MethodPatch:
		for _, value := range h.accounts { // запрашиваем данные формата имя аккаунта значение
			names = append(names, value.Name)
			values = append(values, value.Value)
		} // костыль. Без данного цикла не получается взять записанные в методГете данные
		fmt.Println("patch, values:", values)
		if err := h.PatchAccounts(names); err != nil {
			resp.WriteHeader(http.StatusNotFound)
			_, _ = resp.Write([]byte("such account not found"))
			return
		}
		resp.WriteHeader(http.StatusFound) // если доп условий не встретилось подтверждаем успешность
		_, _ = resp.Write([]byte("accounts have been successfully patched"))
	}

}

func (h *Handler) createAccount(name string, value int) error { // функция создание аккаунта
	h.rwMutex.RLock() // блокируем для невозможности изменения данных с других возможных источников доступа

	_, ok := h.accounts[name] // запрашиваем имя
	if ok {                   // если уже существует, то ok не будет пустым, соотвественно возвращаем ошибку
		h.rwMutex.RUnlock()

		return errAccountAlreadyExist
	}

	h.rwMutex.RUnlock() // пока не работаем с возможным изменением данных, разблокируем до лучших времён

	h.rwMutex.Lock()
	defer h.rwMutex.Unlock() // разблокируем после выполнения всего, что ещё будет выполнено дальше по ходу функции

	account := NewAccount(name, value) // после успешного прохождения всех проверок создали аккаунт

	h.accounts[name] = account

	return nil
}

func (h Handler) GetAccountNames() error {
	h.rwMutex.Lock()
	ok := h.accounts  // запрашиваем все имена
	if len(ok) == 0 { // если не существует, то ok будет пустым, соотвественно возвращаем ошибку
		h.rwMutex.Unlock()
		return errAccountsDontExist
	}
	h.rwMutex.Unlock()
	return nil
}

func (h Handler) DeleteAccounts(names []string) error {
	h.rwMutex.Lock()
	if len(h.accounts) == 0 {
		h.rwMutex.Unlock()
		return errAccountsDontExist
	}

	for _, value := range names { // удаляем поля по полученным значениям
		// Проверяем, существует ли аккаунт с таким именем
		if _, ok := h.accounts[value]; ok {
			delete(h.accounts, value)
		} else {
			// Если аккаунт с таким ключом не существует, можно сделать что-то еще, например, вывести сообщение об ошибке
			return errSuchAccountDontExist
		}

	}
	h.rwMutex.Unlock()
	return nil
}

func (h Handler) PatchAccounts(names []string) error {
	h.rwMutex.Lock()
	if len(h.accounts) == 0 {
		h.rwMutex.Unlock()
		return errAccountsDontExist
	}

	for _, name := range names { // удаляем поля по полученным значениям
		// Проверяем, существует ли аккаунт с таким именем
		if acc, ok := h.accounts[name]; ok {
			acc.Value += 1
		} else {
			// Если аккаунт с таким ключом не существует, можно сделать что-то еще, например, вывести сообщение об ошибке
			return errSuchAccountDontExist
		}
	}
	h.rwMutex.Unlock()
	return nil
}

func main() {
	h := &Handler{ // присваиваем обработчик для поднятого сервера
		accounts: make(map[string]*Account),
		rwMutex:  &sync.RWMutex{},
	}
	fmt.Println(h)
	if err := http.ListenAndServe(":9999", h); !errors.Is(err, http.ErrServerClosed) {
		// ^ запускаем сервер с адресом 9999 и обработчиком h
		//                                          ^ проверяем работоспособность сервера (не упал ли)
		panic(err) // останавливаем выполнение программы
	}
}
