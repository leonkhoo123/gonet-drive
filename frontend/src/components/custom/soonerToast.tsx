import { CircleCheck, CircleX } from 'lucide-react';
import { Toaster, type ToasterProps } from 'sonner';
import { useTheme } from '@/components/theme-provider';

// --- 1. Custom Icon Wrappers for Color ---

// Success Icon styled with 'text-primary' from your shadcn/ui theme
const SuccessIcon = () => (
    <div className="text-green-500 flex items-center justify-center mr-4">
        <CircleCheck size={24} />
    </div>
);

// Error Icon styled with a standard Tailwind red color
const ErrorIcon = () => (
    <div className="text-red-500 flex items-center justify-center mr-4">
        <CircleX size={24} />
    </div>
);

// --- 2. The Custom Toaster Component ---

// Note: We use ToasterProps if you want to pass additional props from App.tsx
export const SonnerToastCustom = (props: ToasterProps) => {
  const { theme } = useTheme();

  return (
    <Toaster 
      // 1. Positioning and Offset
      position="bottom-center" 
      offset={100} // Pushes the toast bar 30px up from the bottom edge
      duration={2500}
      
      // Follow the application theme
      theme={theme as ToasterProps['theme']}

      // 2. Custom Icons for Coloring
      icons={{
        success: <SuccessIcon />,
        error: <ErrorIcon />,
      }}
      
      // 3. Global Toast Styling
      toastOptions={{
        // Common styles for all toast cards (makes them wider, larger padding/font)
        className: 'p-4 rounded-lg shadow-xl w-80 text-lg flex items-center',
        classNames: {
          toast: 'group toast p-4 rounded-lg shadow-xl w-80 text-lg flex items-center',
          // Additional margin for the icon and content as a fallback
          icon: 'mr-4',
          content: 'ml-2',
        },
        style: { 
            gap: '16px', // Ensures a strict gap between flex items
            padding: '16px'
        },
      }}
      
      // 4. Spread any additional props passed from the parent (like theme, invert, etc.)
      {...props}
    />
  );
};