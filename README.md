## 部署
创建应用:
docker-compose up --build -d


关闭应用:
docker-compose down



## 使用说明
参数:
url
filters
selector

例如:
http://localhost:8080/convert?url=https://developers.weixin.qq.com/miniprogram/dev/api/ad/wx.createInterstitialAd.html&filters=.sidebar&filters=.navbar&filters=.subnavbar&filters=.footer&filters=.fixed-translate&selector=.main-container