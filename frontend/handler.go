package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"frontend/money"
	pb "frontend/proto"
)

var log *logrus.Logger

func initializeLogger() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
}

// Initialize logger
func init() {
	initializeLogger()
}

// Home page
func (fe *FrontendServer) HomeHandler(ctx *gin.Context) {
	r := ctx.Request
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search for currencies"), http.StatusInternalServerError)
		return
	}
	products, err := fe.getProducts(r.Context())

	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search for products"), http.StatusInternalServerError)
		return
	}
	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search to cart"), http.StatusInternalServerError)
		return
	}

	type productView struct {
		Item  *pb.Product
		Price *pb.Money
	}
	ps := make([]productView, len(products))
	for i, p := range products {
		price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			renderHTTPError(log, ctx, errors.Wrapf(err, "Failed currency conversion %s", p.GetId()), http.StatusInternalServerError)
			return
		}
		ps[i] = productView{p, price}
	}

	resultMap := map[string]interface{}{
		"session_id":    sessionID(r),
		"request_id":    r.Context().Value(ctxKeyRequestID{}),
		"user_currency": currentCurrency(r),
		"show_currency": true,
		"currencies":    currencies,
		"products":      ps,
		"cart_size":     cartSize(cart),
		"ad":            fe.chooseAd(r.Context(), []string{}, log),
	}

	ctx.HTML(http.StatusOK, "home", resultMap)

}

// Product page
func (fe *FrontendServer) ProductHandler(ctx *gin.Context) {
	r := ctx.Request
	id := ctx.Param("id")
	if id == "" {
		renderHTTPError(log, ctx, errors.New("Product id not specified"), http.StatusBadRequest)
		return
	}
	log.WithField("id", id).WithField("currency", currentCurrency(r)).
		Debug("Product services")

	p, err := fe.getProduct(r.Context(), id)
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search for products"), http.StatusInternalServerError)
		return
	}
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search for currencies"), http.StatusInternalServerError)
		return
	}

	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search to cart"), http.StatusInternalServerError)
		return
	}

	price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Failed currency conversion"), http.StatusInternalServerError)
		return
	}

	recommendations, err := fe.getRecommendations(r.Context(), sessionID(r), []string{id})
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Failed product recommendation"), http.StatusInternalServerError)
		return
	}

	product := struct {
		Item  *pb.Product
		Price *pb.Money
	}{p, price}

	resultMap := map[string]interface{}{
		"session_id":      sessionID(r),
		"request_id":      r.Context().Value(ctxKeyRequestID{}),
		"ad":              fe.chooseAd(r.Context(), p.Categories, log),
		"user_currency":   currentCurrency(r),
		"show_currency":   true,
		"currencies":      currencies,
		"product":         product,
		"recommendations": recommendations,
		"cart_size":       cartSize(cart),
	}

	ctx.HTML(http.StatusOK, "product", resultMap)
}

// Add to cart handler
func (fe *FrontendServer) addToCartHandler(ctx *gin.Context) {
	r := ctx.Request
	w := ctx.Writer
	quantity, _ := strconv.ParseUint(r.FormValue("quantity"), 10, 32)
	productID := r.FormValue("product_id")
	if productID == "" || quantity == 0 {
		renderHTTPError(log, ctx, errors.New("Invalid form input"), http.StatusBadRequest)
		return
	}
	log.WithField("product", productID).WithField("quantity", quantity).Debug("Add to cart")

	p, err := fe.getProduct(r.Context(), productID)
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search for products"), http.StatusInternalServerError)
		return
	}

	if err := fe.insertCart(r.Context(), sessionID(r), p.GetId(), int32(quantity)); err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Conditional Cart Failure"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("location", "/cart")
	w.WriteHeader(http.StatusFound)
}

// Empty cart
func (fe *FrontendServer) emptyCartHandler(ctx *gin.Context) {
	r := ctx.Request
	w := ctx.Writer

	log.Debug("Empty cart")

	if err := fe.emptyCart(r.Context(), sessionID(r)); err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Failed to empty cart"), http.StatusInternalServerError)
		return
	}
	w.Header().Set("location", "/")
	w.WriteHeader(http.StatusFound)
}

// View Cart
func (fe *FrontendServer) viewCartHandler(ctx *gin.Context) {
	r := ctx.Request
	log.Debug("View Cart")
	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search for currencies"), http.StatusInternalServerError)
		return
	}
	cart, err := fe.getCart(r.Context(), sessionID(r))
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search to cart"), http.StatusInternalServerError)
		return
	}

	recommendations, err := fe.getRecommendations(r.Context(), sessionID(r), cartIDs(cart))
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Failed to get recommendation"), http.StatusInternalServerError)
		return
	}

	shippingCost, err := fe.getShippingQuote(r.Context(), cart, currentCurrency(r))
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Failed to shipping"), http.StatusInternalServerError)
		return
	}

	type cartItemView struct {
		Item     *pb.Product
		Quantity int32
		Price    *pb.Money
	}
	items := make([]cartItemView, len(cart))
	totalPrice := &pb.Money{CurrencyCode: currentCurrency(r)}
	for i, item := range cart {
		p, err := fe.getProduct(r.Context(), item.GetProductId())
		if err != nil {
			renderHTTPError(log, ctx, errors.Wrapf(err, "Cannot search for products #%s", item.GetProductId()), http.StatusInternalServerError)
			return
		}
		price, err := fe.convertCurrency(r.Context(), p.GetPriceUsd(), currentCurrency(r))
		if err != nil {
			renderHTTPError(log, ctx, errors.Wrapf(err, "Cannot search for currencies #%s", item.GetProductId()), http.StatusInternalServerError)
			return
		}

		multPrice := money.MultiplySlow(price, uint32(item.GetQuantity()))
		items[i] = cartItemView{
			Item:     p,
			Quantity: item.GetQuantity(),
			Price:    multPrice,
		}
		totalPrice = money.Must(money.Sum(totalPrice, multPrice))
	}
	totalPrice = money.Must(money.Sum(totalPrice, shippingCost))
	year := time.Now().Year()

	resultMap := map[string]interface{}{
		"session_id":       sessionID(r),
		"request_id":       r.Context().Value(ctxKeyRequestID{}),
		"user_currency":    currentCurrency(r),
		"currencies":       currencies,
		"recommendations":  recommendations,
		"cart_size":        cartSize(cart),
		"shipping_cost":    shippingCost,
		"show_currency":    true,
		"total_cost":       totalPrice,
		"items":            items,
		"expiration_years": []int{year, year + 1, year + 2, year + 3, year + 4},
	}

	ctx.HTML(http.StatusOK, "cart", resultMap)

}

// Place order
func (fe *FrontendServer) placeOrderHandler(ctx *gin.Context) {
	r := ctx.Request
	log.Debug("Place order")

	var (
		email         = r.FormValue("email")
		streetAddress = r.FormValue("street_address")
		zipCode, _    = strconv.ParseInt(r.FormValue("zip_code"), 10, 32)
		city          = r.FormValue("city")
		state         = r.FormValue("state")
		country       = r.FormValue("country")
		ccNumber      = strings.ReplaceAll(r.FormValue("credit_card_number"), "-", "")
		ccMonth, _    = strconv.ParseInt(r.FormValue("credit_card_expiration_month"), 10, 32)
		ccYear, _     = strconv.ParseInt(r.FormValue("credit_card_expiration_year"), 10, 32)
		ccCVV, _      = strconv.ParseInt(r.FormValue("credit_card_cvv"), 10, 32)
	)

	order, err := fe.checkoutService.PlaceOrder(r.Context(), &pb.PlaceOrderRequest{
		Email: email,
		CreditCard: &pb.CreditCardInfo{
			CreditCardNumber:          ccNumber,
			CreditCardExpirationMonth: int32(ccMonth),
			CreditCardExpirationYear:  int32(ccYear),
			CreditCardCvv:             int32(ccCVV)},
		UserId:       sessionID(r),
		UserCurrency: currentCurrency(r),
		Address: &pb.Address{
			StreetAddress: streetAddress,
			City:          city,
			State:         state,
			ZipCode:       int32(zipCode),
			Country:       country},
	})
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Failed to place order"), http.StatusInternalServerError)
		return
	}
	log.WithField("order", order.GetOrder().GetOrderId()).Info("Place order")

	order.GetOrder().GetItems()
	recommendations, _ := fe.getRecommendations(r.Context(), sessionID(r), nil)

	totalPaid := order.GetOrder().GetShippingCost()
	for _, v := range order.GetOrder().GetItems() {
		multPrice := money.MultiplySlow(v.GetCost(), uint32(v.GetItem().GetQuantity()))
		totalPaid = money.Must(money.Sum(totalPaid, multPrice))
	}

	currencies, err := fe.getCurrencies(r.Context())
	if err != nil {
		renderHTTPError(log, ctx, errors.Wrap(err, "Cannot search for currencies"), http.StatusInternalServerError)
		return
	}

	resultMap := map[string]interface{}{
		"session_id":      sessionID(r),
		"request_id":      r.Context().Value(ctxKeyRequestID{}),
		"user_currency":   currentCurrency(r),
		"show_currency":   false,
		"currencies":      currencies,
		"order":           order.GetOrder(),
		"total_paid":      &totalPaid,
		"recommendations": recommendations,
	}

	ctx.HTML(http.StatusOK, "order", resultMap)
}

// Log out
func (fe *FrontendServer) logoutHandler(ctx *gin.Context) {
	r := ctx.Request
	w := ctx.Writer
	log.Debug("Log out")
	for _, c := range r.Cookies() {
		c.Expires = time.Now().Add(-time.Hour * 24 * 365)
		c.MaxAge = -1
		http.SetCookie(w, c)
	}
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

// Setting the currency
func (fe *FrontendServer) setCurrencyHandler(ctx *gin.Context) {
	r := ctx.Request
	w := ctx.Writer

	cur := r.FormValue("currency_code")
	log.WithField("curr.new", cur).WithField("curr.old", currentCurrency(r)).
		Debug("setting currency")

	if cur != "" {
		http.SetCookie(w, &http.Cookie{
			Name:   cookieCurrency,
			Value:  cur,
			MaxAge: cookieMaxAge,
		})
	}
	referer := r.Header.Get("referer")
	if referer == "" {
		referer = "/"
	}
	w.Header().Set("Location", referer)
	w.WriteHeader(http.StatusFound)
}

// Close Ads
func (fe *FrontendServer) chooseAd(ctx context.Context, ctxKeys []string, log logrus.FieldLogger) *pb.Ad {
	ads, err := fe.getAd(ctx, ctxKeys)
	if err != nil {
		log.WithField("error", err).Warn("Failed to query ads")
		return nil
	}
	if len(ads) == 0 {
		return nil
	}
	return ads[rand.Intn(len(ads))]
}

// Error message
func renderHTTPError(log logrus.FieldLogger, ctx *gin.Context, err error, code int) {
	r := ctx.Request
	w := ctx.Writer
	log.WithField("error", err).Error("Request Error")
	errMsg := fmt.Sprintf("%+v", err)

	w.WriteHeader(code)

	resultMap := map[string]interface{}{
		"session_id":  sessionID(r),
		"request_id":  r.Context().Value(ctxKeyRequestID{}),
		"error":       errMsg,
		"status_code": code,
		"status":      http.StatusText(code),
	}

	ctx.HTML(http.StatusOK, "error", resultMap)
}

// current currency
func currentCurrency(r *http.Request) string {
	c, _ := r.Cookie(cookieCurrency)
	if c != nil {
		return c.Value
	}
	return defaultCurrency
}

// session
func sessionID(r *http.Request) string {
	v := r.Context().Value(ctxKeySessionID{})
	if v != nil {
		return v.(string)
	}
	return ""
}

// cartIDs
func cartIDs(c []*pb.CartItem) []string {
	out := make([]string, len(c))
	for i, v := range c {
		out[i] = v.GetProductId()
	}
	return out
}

// cart size
func cartSize(c []*pb.CartItem) int {
	cartSize := 0
	for _, item := range c {
		cartSize += int(item.GetQuantity())
	}
	return cartSize
}

// render currency
func renderMoney(money *pb.Money) string {
	currencyLogo := renderCurrencyLogo(money.GetCurrencyCode())
	return fmt.Sprintf("%s%d.%02d", currencyLogo, money.GetUnits(), money.GetNanos()/10000000)
}

// render currency logo
func renderCurrencyLogo(currencyCode string) string {
	logos := map[string]string{
		"USD": "$",
		"CAD": "$",
		"JPY": "¥",
		"EUR": "€",
		"TRY": "₺",
		"GBP": "£",
	}

	logo := "$" //default
	if val, ok := logos[currencyCode]; ok {
		logo = val
	}
	return logo
}

// Determine if a string is in a string slice
func stringinSlice(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
