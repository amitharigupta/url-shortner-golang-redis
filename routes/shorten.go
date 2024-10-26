package routes


import (
	"time"
	"os"
	"strconv"
	"github.com/amitharigupta/url-shortner-golang-redis/database"
	"github.com/amitharigupta/url-shortner-golang-redis/helpers"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/asaskevich/govalidator"
	"github.com/google/uuid"
)

type request struct {
	URL string `json:"url"`
	CustomShort string `json:"custom_short"`
	Expiry time.Duration	`json:"expiry"`
}

type response struct {
    URL	string `json:"url"`
	CustomShort string `json:"custom_short"`
	Expiry time.Duration `json:"expiry"`
	XRateRemaining int `json:"x_rate_remaining"`
	XRateLimitReset time.Duration `json:"x_rate_limit_rest"`
}


func ShortenURL (c *fiber.Ctx) error {
	
	body := new(request)

	if err := c.BodyParser(body); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Cannot Parse JSON",
        })
    }

	// implement rate limiting

	r2 := database.CreateClient(1)
	defer r2.Close()
	val, err := r2.Get(database.Ctx, c.IP()).Result()

	if err == redis.Nil {
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30	* 60 * time.Second).Err()
	} else {
		val, _ = r2.Get(database.Ctx, c.IP()).Result()
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {

			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
            return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
                "error": "Rate limit exceeded",
				"rate_limit_rest": limit / time.Nanosecond / time.Minute,
            })
        }
	}

	// implement custom shortening logic
    // implement URL expiry

    // generate a unique short URL
    // customShort := generateCustomShort()

	// check if the input is an actual URL

	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid URL",
        })
	}

	// check for domain error

	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Domain error",
        })
	}

	// enforce https, SSL

	body.URL = helpers.EnforceHTTP(body.URL)

	var id string

	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()

	val, _ = r.Get(database.Ctx, id).Result()
	if val != "" {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error": "Short URL already exists",
        })
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry).Err()

	if err!= nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Error while storing in database",
        })
    }


	resp := response {
		URL: body.URL,
        CustomShort: "",
        Expiry: body.Expiry,
        XRateRemaining: 10, // replace with actual value
        XRateLimitReset: 30, // replace with actual value
	}

	r2.Decr(database.Ctx, c.IP())


	val, _ = r2.Get(database.Ctx, c.IP()).Result()
	resp.XRateRemaining, _ = strconv.Atoi(val)


	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl 

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id

	return c.Status(fiber.StatusOK).JSON(resp)
}