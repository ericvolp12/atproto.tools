import "./App.css";

import {
  createBrowserRouter,
  Link,
  RouterProvider,
  useLocation,
} from "react-router-dom";

import Records from "./components/records-temp/Records";
import Welcome from "./components/welcome/Welcome";
import LexiconEditor from "./components/lexicon-editor/LexiconEditor";

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
    <header className="fixed top-0 z-50 w-full bg-white text-center text-slate-600 dark:bg-slate-900 dark:text-slate-200">
      <div className="mx-auto max-w-7xl px-2 align-middle ">
        <span className="footer-text text-xs">
          <Link to="/" className={active(["/"])}>
            welcome
          </Link>
          {" | "}
          <Link to="/records" className={active(["/records"])}>
            records
          </Link>
          {" | "}
          <Link to="/lexicons" className={active(["/lexicons"])}>
            lexicons
          </Link>
          {" | "}
          <Link
            to="https://github.com/ericvolp12/atproto.tools"
            target="_blank"
            className="my-auto inline-block h-3 w-3 fill-slate-600 dark:fill-slate-200"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="16"
              height="16"
              viewBox="0 0 24 24"
            >
              <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z" />
            </svg>
          </Link>
        </span>
      </div>
    </header>
  );
};

const ThemeWrapper: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  return (
    <div className="min-h-dvh bg-white text-slate-600 dark:bg-slate-900 dark:text-slate-200">
      {children}
    </div>
  );
};

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
    ),
  },
  {
    path: "/lexicons",
    element: (
      <ThemeWrapper>
        <NavList />
        <LexiconEditor />
      </ThemeWrapper>
    ),
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
