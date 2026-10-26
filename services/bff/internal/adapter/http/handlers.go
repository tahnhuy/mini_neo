package httpadapter

import (
	orderv1 "github.com/tahnhuy/mini_neo/gen/go/mini/order/v1"
	stockv1 "github.com/tahnhuy/mini_neo/gen/go/mini/stock/v1"
	"net/http"
)

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	msg, err := s.order.ListOrders(r.Context(), &orderv1.ListOrdersRequest{})
	respond(w, msg, err)
}
func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	msg, err := s.order.GetOrder(r.Context(), &orderv1.GetOrderRequest{Id: r.PathValue("id")})
	respond(w, msg, err)
}
func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	msg, err := s.order.ApproveOrder(r.Context(), &orderv1.ApproveOrderRequest{Id: r.PathValue("id")})
	respond(w, msg, err)
}
func (s *Server) items(w http.ResponseWriter, r *http.Request) {
	msg, err := s.stock.ListItems(r.Context(), &stockv1.ListItemsRequest{})
	respond(w, msg, err)
}
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	req := &orderv1.CreateOrderRequest{}
	if err := decode(w, r, req); err != nil {
		writeError(w, err)
		return
	}
	msg, err := s.order.CreateOrder(r.Context(), req)
	respond(w, msg, err)
}
func (s *Server) reject(w http.ResponseWriter, r *http.Request) {
	req := &orderv1.RejectOrderRequest{}
	if err := decode(w, r, req); err != nil {
		writeError(w, err)
		return
	}
	req.Id = r.PathValue("id")
	msg, err := s.order.RejectOrder(r.Context(), req)
	respond(w, msg, err)
}
