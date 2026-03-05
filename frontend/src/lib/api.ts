import axios, { AxiosInstance } from 'axios'

const BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1'

class ApiClient {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: BASE_URL,
      headers: { 'Content-Type': 'application/json' },
    })

    this.client.interceptors.request.use((config) => {
      const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null
      if (token) config.headers.Authorization = `Bearer ${token}`
      return config
    })

    this.client.interceptors.response.use(
      (res) => res,
      (err) => {
        if (err.response?.status === 401) {
          localStorage.removeItem('token')
          window.location.href = '/auth/login'
        }
        return Promise.reject(err)
      }
    )
  }

  // Auth
  register = (data: { name: string; email: string; password: string }) =>
    this.client.post('/auth/register', data)
  login = (data: { email: string; password: string }) =>
    this.client.post('/auth/login', data)
  me = () => this.client.get('/auth/me')

  // Listings
  getListings = (params?: Record<string, string | number>) =>
    this.client.get('/listings', { params })
  getListing = (id: string) => this.client.get(`/listings/${id}`)
  createListing = (data: FormData | object) => this.client.post('/listings', data)
  updateListing = (id: string, data: object) => this.client.put(`/listings/${id}`, data)
  deleteListing = (id: string) => this.client.delete(`/listings/${id}`)
  toggleFavorite = (id: string) => this.client.post(`/listings/${id}/favorite`)
  getMyListings = () => this.client.get('/listings/me')
  getCategories = () => this.client.get('/categories')

  // Auctions
  getAuctions = (params?: object) => this.client.get('/auctions', { params })
  getAuction = (id: string) => this.client.get(`/auctions/${id}`)
  createAuction = (data: object) => this.client.post('/auctions', data)
  placeBid = (id: string, data: { amount: number }) => this.client.post(`/auctions/${id}/bid`, data)
  getAuctionBids = (id: string) => this.client.get(`/auctions/${id}/bids`)

  // Chat
  getConversations = () => this.client.get('/chat/conversations')
  createConversation = (data: { other_user_id: string; listing_id?: string }) =>
    this.client.post('/chat/conversations', data)
  getMessages = (convId: string) => this.client.get(`/chat/conversations/${convId}/messages`)
  sendMessage = (convId: string, data: { content: string; type?: string }) =>
    this.client.post(`/chat/conversations/${convId}/messages`, data)

  // Payments
  getStripeKey = () => this.client.get('/payments/key')
  createPaymentIntent = (data: { amount: number; currency?: string; listing_id?: string }) =>
    this.client.post('/payments/intent', data)
}

export const api = new ApiClient()
export default api
