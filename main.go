package main

import (
    "github.com/gin-gonic/gin"
    "github.com/ulule/limiter/v3"
    gin2 "github.com/ulule/limiter/v3/drivers/middleware/gin"
    "github.com/ulule/limiter/v3/drivers/store/memory"
    "html/template"
    "log"
    "math/rand"
    "net/http"
    "strconv"
    "strings"
    "sync"
    "time"
)

const RoomIDLength = 32
const RateLimitString = "5-S"
const DeadRoomCleanupRate = 2 * time.Hour

type Room struct {
    entries []string
    lastUsed time.Time
}

type RoomMap struct {
    rooms map[string]Room
    mutex sync.Mutex
}

type TemplateContext struct {
    ID  string
}

var Rooms = NewRoomMap()

func main() {
    rand.Seed(time.Now().Unix())

    router := gin.Default()
    addRateLimit(router)
    go cleanDeadRooms()

    router.GET("/", displayIndex)
    router.GET("/add/:roomID", displayAddPage)
    router.GET("/draw/:roomID", displayDrawPage)
    router.GET("/error", displayErrorPage)
    apiRoutes := router.Group("/api")
    {
        apiRoutes.GET("/create", apiCreateRoom)
        apiRoutes.POST("/put", apiPutEntry)
        apiRoutes.GET("/count/:roomID", apiGetCount)
        apiRoutes.GET("/draw/:roomID", apiDrawEntry)
    }

    router.Run()
}

func addRateLimit(g *gin.Engine) {
    rate, err := limiter.NewRateFromFormatted(RateLimitString)
    if err != nil {
        log.Fatal(err)
    }
    store := memory.NewStore()
    instance := limiter.New(store, rate)
    middleware := gin2.NewMiddleware(instance)
    g.Use(middleware)
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

    tmp, err := template.ParseFiles("./templates/add.html")
    if err != nil {
        ctx.Redirect(http.StatusFound, "/error")
        return
    }
    tmp.Execute(ctx.Writer, NewTemplateContext(id))
}

func displayDrawPage(ctx *gin.Context) {
    Rooms.mutex.Lock()
    defer Rooms.mutex.Unlock()

    id := ctx.Param("roomID")
    if !roomExists(id) {
        ctx.Redirect(http.StatusFound, "/error")
        return
    }

    tmp, err := template.ParseFiles("./templates/draw.html")
    if err != nil {
        ctx.Redirect(http.StatusFound, "/error")
        return
    }
    tmp.Execute(ctx.Writer, NewTemplateContext(id))
}

func apiCreateRoom(ctx *gin.Context) {
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

func apiPutEntry(ctx *gin.Context) {
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

func apiGetCount(ctx *gin.Context) {
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

func apiDrawEntry(ctx *gin.Context) {
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

func cleanDeadRooms() {
    for true {
        Rooms.mutex.Lock()

        deletions := make([]string, 0)
        for k, v := range Rooms.rooms {
            if time.Now().Sub(v.lastUsed) >= DeadRoomCleanupRate {
                deletions = append(deletions, k)
            }
        }

        for _, k := range deletions {
            delete(Rooms.rooms, k)
            log.Printf("deleted %s\n", k)
        }
        Rooms.mutex.Unlock()
        time.Sleep(1 * time.Minute)
    }
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

func NewTemplateContext(ID string) TemplateContext {
    return TemplateContext{
        ID:  ID,
    }
}