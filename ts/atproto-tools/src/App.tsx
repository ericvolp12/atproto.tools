import "./App.css";
import {
  Link,
  RouterProvider,
  createBrowserRouter,
  useLocation,
} from "react-router-dom";
import Welcome from "./components/welcome/Welcome";
import Records from "./components/records-temp/Records";


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
    <header className="fixed top-0 text-center w-full z-50 bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-200">
      <div className="mx-auto max-w-7xl px-2 align-middle ">
        <span className="footer-text text-xs">
          <Link to="/" className={active(["/"])}>
            welcome
          </Link>
          {" | "}
          <Link to="/records" className={active(["/records"])}>
            records
          </Link>
        </span>
      </div>
    </header>
  );
};

const ThemeWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <div className="bg-white dark:bg-slate-900 text-slate-600 dark:text-slate-200 min-h-screen">
      {children}
    </div>
  );
}

const router = createBrowserRouter([
  {
    path: "/",
    element: (
      <ThemeWrapper>
        <NavList />
        <Welcome />
      </ThemeWrapper>
    ),
  },
  {
    path: "/records",
    element: (
      <ThemeWrapper>
        <NavList />
        <Records />
      </ThemeWrapper>
    )
  }
]);

function App() {
  return (
    <>
      <RouterProvider router={router} />
    </>
  );
}

export default App;
