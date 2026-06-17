import { BrowserRouter, Navigate, Outlet, Route, Routes } from "react-router-dom";
import { AuthProvider, useAuth } from "./features/auth/AuthContext";
import LoginPage from "./features/auth/LoginPage";
import SignupPage from "./features/auth/SignupPage";
import ProductDetailPage from "./features/products/ProductDetailPage";
import SellPage from "./features/products/SellPage";
import ChatPage from "./features/chat/ChatPage";
import Home from "./pages/Home";
import Likes from "./pages/Likes";
import MyPage from "./pages/MyPage";
import AuctionDetailPage from "./pages/AuctionDetailPage";
import BottomNav from "./components/BottomNav";
import "./App.css";

function PrivateRoute({ children }: { children: JSX.Element }) {
  const { user, isLoading } = useAuth();
  if (isLoading) return null;
  return user ? children : <Navigate to="/login" replace />;
}

function Layout() {
  return (
    <>
      <Outlet />
      <BottomNav />
    </>
  );
}

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          {/* BottomNav付きレイアウト */}
          <Route element={<Layout />}>
            <Route path="/" element={<Home />} />
            <Route path="/likes" element={<Likes />} />
            <Route path="/mypage" element={<MyPage />} />
            <Route path="/products/:id" element={<ProductDetailPage />} />
            <Route path="/auctions/:id" element={<AuctionDetailPage />} />
            <Route
              path="/sell"
              element={
                <PrivateRoute>
                  <SellPage />
                </PrivateRoute>
              }
            />
          </Route>

          {/* BottomNavなし */}
          <Route path="/login" element={<LoginPage />} />
          <Route path="/signup" element={<SignupPage />} />
          <Route
            path="/products/:id/chat"
            element={
              <PrivateRoute>
                <ChatPage />
              </PrivateRoute>
            }
          />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
