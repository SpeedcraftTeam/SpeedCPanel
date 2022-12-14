package main

import (
	"SpeedCPanelManager/schema"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
)

type ServerCreateParams struct {
	Name      string `json:"name"`
	Hostname  string `path:"hostname"`
	NetworkId int    `path:"networkID"`
	Version   string `json:"version"`
	Image     string `head:"X-Docker-Image"`
	Type      string `json:"type"`
	Premium   bool   `json:"premium"`
	Modpack   string `json:"modpack"`
}

func createContainer(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	if err := user.Claims.Valid(); err == nil {
		var params ServerCreateParams
		c.Bind(&params)
		var network schema.Network
		if err != nil {
			return err
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		res := db.Database(config.DB.Database).Collection("Networks").FindOne(timeoutCtx, bson.D{{
			"_id",
			params.NetworkId,
		}})
		if err = res.Decode(network); err != nil {
			return err
		}
		env := []string{"EULA=TRUE", fmt.Sprintf("VERSION=%s", params.Version, fmt.Sprintf("TYPE=%s", params.Type))}
		if params.Type == "CURSEFORGE" {
			env = append(env, fmt.Sprintf("CF_SERVER_MOD=%s", params.Modpack))
		} else if params.Type == "FTBA" {
			env = append(env, fmt.Sprintf("FTB_MODPACK_ID=%s", params.Modpack))
		}
		result, err := client.ServiceCreate(timeoutCtx, swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: params.Name,
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image:    config.Images[params.Image],
					Env:      env,
					Hostname: params.Hostname,
					Labels: map[string]string{
						"traefik.tcp.routers.mc.rule": "HostSNI(`*`)",
						"traefik.port":                strconv.Itoa(25565 + len(network.Containers)),
					},
				},
			},
			Networks: []swarm.NetworkAttachmentConfig{swarm.NetworkAttachmentConfig{Target: network.DockerID}},
		}, types.ServiceCreateOptions{})
		if err != nil {
			return err
		}
		return c.JSON(http.StatusAccepted, result)
	} else {
		return err
	}
}
