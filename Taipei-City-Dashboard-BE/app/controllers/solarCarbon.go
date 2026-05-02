package controllers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/tpe-doit/Taipei-City-Dashboard-BE/app/utils"
)

func EstimateSolarCarbon(c *gin.Context) {
    var input utils.SolarInput

    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "輸入格式錯誤：" + err.Error(),
        })
        return
    }

    if input.Area <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "面積必須大於 0"})
        return
    }

    result, err := utils.CalcSolarCarbon(input)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, result)
}