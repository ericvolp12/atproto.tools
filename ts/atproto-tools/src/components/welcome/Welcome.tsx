import { FC, useEffect } from "react";

const Welcome: FC<{}> = () => {
    useEffect(() => {
        document.title = "Welcome to ATProto Tools";
    }, []);

    return (
        <div className="bg-white mt-10">
            <div className="mx-auto max-w-7xl px-2 align-middle">
                <h1 className="text-4xl font-bold">Welcome to AT Proto Tools</h1>
                <p className="text-lg">Coming Soon: a suite of tools to help you build and test AT Proto applications and infrastructure.</p>
            </div>
        </div >
    );
};

export default Welcome;
