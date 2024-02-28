import "./App.css";
import {
  Link,
  RouterProvider,
  createBrowserRouter,
  useLocation,
} from "react-router-dom";
import Welcome from "./components/welcome/Welcome";


const NavList: React.FC = () => {
  let location = useLocation();

  const inactive = "font-bold underline-offset-1 underline opacity-50";

  const active = (path: string[]) => {
    let isActive = false;
    if (path.some((p) => location.pathname === p)) {
      isActive = true;
    }

    return isActive ? "font-bold underline-offset-1 underline" : inactive;
  };

  return (
    <header className="bg-white fixed top-0 text-center w-full z-50">
      <div className="mx-auto max-w-7xl px-2 align-middle">
        <span className="footer-text text-xs">
          <Link to="/" className={active(["/"])}>
            welcome
          </Link>
        </span>
      </div>
    </header>
  );
};

const router = createBrowserRouter([
  {
    path: "/",
    element: (
      <>
        <NavList />
        <Welcome />
      </>
    ),
  },
]);

function App() {
  return (
    <>
      <RouterProvider router={router} />
    </>
  );
}

export default App;
