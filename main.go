package main

import (
    "github.com/gin-gonic/gin"
    "io/ioutil"
    "math/rand"
    "net/http"
    "strconv"
    "strings"
    "sync"
    "time"
)

const RoomIDLength = 32

type Room struct {
    entries []string
    lastUsed time.Time
}

type RoomMap struct {
    rooms map[string]Room
    mutex sync.Mutex
}

var Rooms = NewRoomMap()

func main() {
    rand.Seed(time.Now().Unix())
    router := gin.Default()

    router.GET("/", displayIndex)
    router.GET("/add/:roomID", displayAddPage)
    router.GET("/draw/:roomID", displayDrawPage)
    router.GET("/error", displayErrorPage)
    apiRoutes := router.Group("/api")
    {
        apiRoutes.GET("/create", createRoom)
        apiRoutes.POST("/put", putEntry)
        apiRoutes.GET("/count/:roomID", getCount)
        apiRoutes.GET("/draw/:roomID", drawEntry)
    }

    router.Run()
}

func displayIndex(ctx *gin.Context) {
    ctx.File("./templates/index.html")
}

func displayErrorPage(ctx *gin.Context) {
    ctx.File("./templates/error.html")
}

func displayAddPage(ctx *gin.Context) {
    Rooms.mutex.Lock()
    defer Rooms.mutex.Unlock()

    id := ctx.Param("roomID")
    if !roomExists(id) {
        ctx.Redirect(http.StatusFound, "/error")
        return
    }

    content, err := ioutil.ReadFile("./templates/add.html")
    if err != nil {
        ctx.Redirect(http.StatusFound, "/error")
        return
    }
    html := string(content)
    html = strings.ReplaceAll(html, "%ID%", id)
    ctx.Writer.Write([]byte(html))
}

func displayDrawPage(ctx *gin.Context) {
    Rooms.mutex.Lock()
    defer Rooms.mutex.Unlock()

    id := ctx.Param("roomID")
    if !roomExists(id) {
        ctx.Redirect(http.StatusFound, "/error")
        return
    }

    content, err := ioutil.ReadFile("./templates/draw.html")
    if err != nil {
        ctx.Redirect(http.StatusFound, "/error")
        return
    }
    html := string(content)
    html = strings.ReplaceAll(html, "%ID%", id)
    ctx.Writer.Write([]byte(html))
}

func createRoom(ctx *gin.Context) {
    Rooms.mutex.Lock()
    defer Rooms.mutex.Unlock()

    id := getRandomId(RoomIDLength)
    for roomExists(id) {
        id = getRandomId(RoomIDLength)
    }

    r := Room{
        entries:  make([]string, 0),
        lastUsed: time.Now(),
    }

    Rooms.rooms[id] = r
    ctx.Redirect(http.StatusFound, "/add/" + id)
}

func putEntry(ctx *gin.Context) {
    Rooms.mutex.Lock()
    defer Rooms.mutex.Unlock()

    id := ctx.GetHeader("roomID")
    entry := ctx.GetHeader("entry")

    if !roomExists(id) {
        ctx.Status(http.StatusBadRequest)
        return
    }

    room := Rooms.rooms[id]
    room.entries = append(room.entries, entry)
    room.lastUsed = time.Now()
    Rooms.rooms[id] = room
    ctx.Status(http.StatusOK)
}

func getCount(ctx *gin.Context) {
    Rooms.mutex.Lock()
    defer Rooms.mutex.Unlock()

    id := ctx.Param("roomID")
    if !roomExists(id) {
        ctx.Status(http.StatusBadRequest)
        return
    }

    room := Rooms.rooms[id]
    room.lastUsed = time.Now()
    Rooms.rooms[id] = room

    resp := strconv.FormatInt(int64(len(room.entries)), 10)
    ctx.String(http.StatusOK, resp)
}

func drawEntry(ctx *gin.Context) {
    Rooms.mutex.Lock()
    defer Rooms.mutex.Unlock()
    id := ctx.Param("roomID")

    if !roomExists(id) {
        ctx.Status(http.StatusBadRequest)
        return
    }


    room := Rooms.rooms[id]

    if len(room.entries) == 0 {
        ctx.String(http.StatusOK, "")
        return
    }

    index := rand.Intn(len(room.entries))
    entry := room.entries[index]
    room.entries = removeFromSlice(room.entries, index)
    room.lastUsed = time.Now()
    Rooms.rooms[id] = room

    ctx.String(http.StatusOK, entry)
}

func roomExists(id string) bool {
    _, isPresent := Rooms.rooms[id]
    return isPresent
}

func getRandomId(idLength int) string {
    const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqurstuvwxyz0123456789"
    const length = len(charset)
    sb := strings.Builder{}

    for i := 0; i < idLength; i++ {
        index := rand.Intn(length)
        sb.WriteString(string(charset[index]))
    }

    return sb.String()
}

func removeFromSlice(slice []string, index int) []string {
    l := len(slice)
    if index < 0 || index >= l {
        return slice
    }

    out := make([]string, l - 1)
    newIndex := 0
    for i, value := range slice {
        if i == index {
            continue
        }
        out[newIndex] = value
        newIndex++
    }

    return out
}

func NewRoomMap() RoomMap {
    return RoomMap{rooms: make(map[string]Room)}
}