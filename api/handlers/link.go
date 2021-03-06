package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	uuid "github.com/satori/go.uuid"
	"github.com/vvatelot/url-shortener/api/entities"
	"github.com/vvatelot/url-shortener/api/repositories"
	"github.com/vvatelot/url-shortener/config"
	"github.com/vvatelot/url-shortener/utils"
)

func GetLinks(c *fiber.Ctx) error {
	var response entities.APIListResponse[entities.Link]

	size, err := strconv.Atoi(c.Query("size"))
	if err != nil {
		size = 10
	}

	page, err := strconv.Atoi(c.Query("page"))
	if err != nil {
		page = 1
	}

	links, err := repositories.ListLinks(size, page)
	if err != nil {
		return c.Status(http.StatusNotFound).SendString("Not found")
	}

	countAllLinks := repositories.CountAllLinks()

	response.Data = links
	response.Pagination = utils.GetPagination(countAllLinks, page, size)

	return c.Status(http.StatusOK).JSON(response)
}

func GetLink(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Invalid id")
	}

	link, err := repositories.GetLinkByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).SendString("Not found")
	}

	return c.Status(http.StatusOK).JSON(link)
}

func AddLink(c *fiber.Ctx) error {
	var link entities.Link

	if err := c.BodyParser(&link); err != nil {
		return c.Status(http.StatusBadRequest).SendString("Invalid request")
	}

	response, err := http.Get(link.URL)
	if err != nil || response.StatusCode != http.StatusOK {
		return c.Status(http.StatusBadRequest).SendString("Invalid URL")
	}
	defer response.Body.Close()

	if title, ok := utils.GetHtmlTitle(response.Body); ok {
		link.Title = title
	} else {
		return c.Status(http.StatusBadRequest).SendString("Can not get Page title")
	}

	link.Key = uuid.NewV4().String()

	result := config.Database.Create(&link)

	if result.Error != nil {
		return c.Status(http.StatusInternalServerError).SendString("Error while saving link")
	}

	return c.Status(http.StatusCreated).JSON(link)
}

func UpdateLink(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString("Invalid id")
	}
	var bodyLink entities.Link

	if err := c.BodyParser(&bodyLink); err != nil {
		return c.Status(http.StatusBadRequest).SendString("Invalid request")
	}

	dbLink, err := repositories.GetLinkByID(id)
	if err != nil {
		return c.Status(http.StatusNotFound).SendString("Not found")
	}

	if bodyLink.Title != "" {
		dbLink.Title = bodyLink.Title
	}

	if bodyLink.URL != "" {
		dbLink.URL = bodyLink.URL
	}

	config.Database.Save(&dbLink)

	return c.Status(http.StatusOK).JSON(dbLink)
}

func DeleteLink(c *fiber.Ctx) error {
	id := c.Params("id")

	result := config.Database.Delete(&entities.Link{}, id)

	if result.RowsAffected == 0 {
		return c.Status(http.StatusNotFound).SendString("Not found")
	}

	return c.Status(http.StatusNoContent).SendString("")
}

func ActivateLink(c *fiber.Ctx) error {
	id := c.Params("id")

	var link entities.Link

	result := config.Database.Find(&link, id)
	link.Active = !link.Active

	if result.RowsAffected == 0 {
		return c.Status(http.StatusNotFound).SendString("Not found")
	}

	config.Database.Save(&link)

	return c.Status(http.StatusOK).JSON(link)
}

func Redirect(c *fiber.Ctx) error {
	key := c.Params("key")
	var link entities.Link

	resultFind := config.Database.Find(&link, "key = ?", key)

	if resultFind.RowsAffected == 0 {
		return c.Status(http.StatusNotFound).SendString("Not found")
	}

	response, _ := http.Get(link.URL)
	click := entities.Click{LinkID: int(link.ID), ResponseCode: response.StatusCode}

	resultCreate := config.Database.Create(&click)

	if resultCreate.Error != nil {
		return c.Status(http.StatusInternalServerError).SendString("Internal server error")
	}

	if !link.Active {
		return c.Status(http.StatusGone).SendString("Link is disabled")
	}

	return c.Redirect(link.URL)
}
