import DefaultLayout from "@/layouts/DefaultLayout";
import { useEffect } from 'react';
import { useNavigate } from "react-router-dom";


// --- Main Component ---
export default function HomePage() {
  const navigate = useNavigate();

  useEffect(() => {
    navigate("/home");
  }, [navigate]);

  // --- Main Layout Render ---
  return (
    <DefaultLayout>
      <div>Navigating to Home</div>
    </DefaultLayout>

  );
}
